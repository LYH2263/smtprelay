package smtprelay

import (
	"context"
	"testing"
	"time"

	"github.com/LYH2263/go-smtprelay/internal/smtpdial"
)

// newTestRelay builds a relay with a successful lab dialer and fixed target.
func newTestRelay(t *testing.T, dialer smtpdial.Dialer) *Relay {
	t.Helper()
	r, err := New(Options{
		Dialer:      dialer,
		DirectHost:  "127.0.0.1",
		DirectPort:  2525,
		MaxAttempts: 3,
		Now:         func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r
}

func basicEnvelope() *Envelope {
	return &Envelope{
		MailFrom:   "from@example.com",
		Recipients: []Recipient{{Address: "to@example.com"}},
		RawBody:    []byte("hello\r\n"),
	}
}

// Submit must accept an envelope whose caller left Headers nil and still store
// a usable map: delivery stamps X-Relay-Attempt / DKIM into it.
func TestSubmitInitializesNilHeaders(t *testing.T) {
	r := newTestRelay(t, &smtpdial.MemoryDialer{Code: 250, Response: "OK"})
	env := basicEnvelope()
	if env.Headers != nil {
		t.Fatalf("precondition: Headers should be nil for this test")
	}

	id, err := r.Submit(env)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	stored, err := r.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored.Headers == nil {
		t.Fatalf("stored Headers is nil; delivery would panic on stamp")
	}
	if got, want := stored.Headers["Message-Id"], "<"+id+"@local.relay>"; got != want {
		t.Fatalf("Message-Id = %q, want %q", got, want)
	}

	// Delivery must not panic; on success Ack removes the message from the
	// queue entirely (Sent counts surviving sent/acked messages, so the right
	// assertion here is that the queue is empty, not stranded inflight).
	if _, err := r.DeliverOnce(context.Background()); err != nil {
		t.Fatalf("DeliverOnce: %v", err)
	}
	if got := r.Stats().InFlight; got != 0 {
		t.Fatalf("InFlight = %d, want 0 (message stranded)", got)
	}
	if got := r.Stats().Total; got != 0 {
		t.Fatalf("Total = %d, want 0 (message should be Acked out of the queue)", got)
	}
}

// Even if a nil-Headers envelope reaches delivery (e.g. restored from an older
// snapshot), the stamp path must allocate rather than panic.
func TestDeliverOnceDoesNotPanicOnNilHeaders(t *testing.T) {
	r := newTestRelay(t, &smtpdial.MemoryDialer{Code: 250, Response: "OK"})
	id, err := r.Submit(basicEnvelope())
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Simulate an older snapshot payload: yank Headers back to nil in place.
	r.mu.Lock()
	r.byID[id].Headers = nil
	r.mu.Unlock()

	// Should not panic; should still deliver and not leave it inflight.
	if _, err := r.DeliverOnce(context.Background()); err != nil {
		t.Fatalf("DeliverOnce: %v", err)
	}
	if got := r.Stats().InFlight; got != 0 {
		t.Fatalf("InFlight = %d, want 0 after recovery", got)
	}
}

// A panic anywhere between markInFlight and Ack/Nack must restore the message
// to pending so it is neither stranded inflight nor silently dropped.
func TestDeliverOnceRecoversPanicRestoresPending(t *testing.T) {
	// Dialer panics mid-send: DeliverOnce's recover must restore to pending.
	r := newTestRelay(t, panicDialer{})
	id, err := r.Submit(basicEnvelope())
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	if _, err := r.DeliverOnce(context.Background()); err == nil {
		t.Fatalf("DeliverOnce: expected error from panicking dialer")
	}

	stored, err := r.Get(id)
	if err != nil {
		t.Fatalf("Get after panic: %v", err)
	}
	if stored.State != StatePending {
		t.Fatalf("State = %s, want pending (restored after panic)", stored.State)
	}
	// And Peek must be able to surface it again.
	if _, err := r.Peek(); err != nil {
		t.Fatalf("Peek after recovery: %v (message is stranded)", err)
	}
}

type panicDialer struct{}

func (panicDialer) Send(context.Context, smtpdial.Request) (int, string, error) {
	panic("boom from dialer")
}
