package smtprelay

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LYH2263/go-smtprelay/internal/metrics"
	"github.com/LYH2263/go-smtprelay/internal/persist"
)

// failingStore is a persister whose Save always fails. It stands in for an
// unwritable snapshot path so the test is deterministic across platforms
// (filesystem-permission failures are flaky on Windows).
type failingStore struct{ saved int }

func (f *failingStore) Load() (persist.Snapshot, error) { return persist.Snapshot{}, nil }
func (f *failingStore) Save(ctx context.Context, snap persist.Snapshot) error {
	f.saved++
	return errors.New("disk unwritable")
}
func (f *failingStore) Close() error { return nil }

// passingStore records the last snapshot so the success path can be asserted.
type passingStore struct{ last persist.Snapshot }

func (p *passingStore) Load() (persist.Snapshot, error) { return persist.Snapshot{}, nil }
func (p *passingStore) Save(ctx context.Context, snap persist.Snapshot) error {
	p.last = snap
	return nil
}
func (p *passingStore) Close() error { return nil }

func newAckTestRelay() (*Relay, string) {
	r := &Relay{
		opts:    Options{MaxQueue: 1024, MaxAttempts: 8, DefaultBackoff: time.Second, Now: time.Now},
		byID:    make(map[string]*Envelope),
		order:   make([]string, 0, 8),
		metrics: metrics.New(),
	}
	env := &Envelope{
		ID:         "msg-00001",
		MailFrom:   "sender@example.com",
		Recipients: []Recipient{{Address: "rcpt@example.com"}},
		RawBody:    []byte("hello"),
		State:      StatePending,
	}
	r.byID[env.ID] = env
	r.order = append(r.order, env.ID)
	return r, env.ID
}

func contains(slice []string, s string) bool {
	for _, x := range slice {
		if x == s {
			return true
		}
	}
	return false
}

// Regression: when persistence fails, Ack must NOT dequeue the message.
// Before the fix, Ack deleted the message from memory before persisting, so a
// persist failure left the queue empty in memory while the on-disk snapshot
// never recorded the ack — a silent loss of delivered-but-undurable mail.
func TestAckPersistFailureKeepsMessageQueued(t *testing.T) {
	r, id := newAckTestRelay()
	r.store = &failingStore{}

	err := r.Ack(id)
	if err == nil {
		t.Fatal("Ack with failing store: want error, got nil")
	}
	if !errors.Is(err, ErrPersist) {
		t.Fatalf("Ack with failing store: want ErrPersist, got %v", err)
	}

	// The message must NOT have left memory: it stays queued for re-ack so a
	// later retry recovers it instead of silently dropping delivered mail.
	if _, ok := r.byID[id]; !ok {
		t.Fatalf("Ack persist failure dropped message from byID; want it retained")
	}
	if !contains(r.order, id) {
		t.Fatalf("Ack persist failure removed id from order; want it retained")
	}

	// State restored to pending so Peek re-selects it for delivery.
	if got := r.byID[id].State; got != StatePending {
		t.Fatalf("state after failed ack = %q, want %q", got, StatePending)
	}

	// Acked counter must not advance — the message was not durably acked.
	if got := r.metrics.Snapshot().Acked; got != 0 {
		t.Fatalf("Acked counter = %d, want 0 (not durably acked)", got)
	}

	// Recovery: once persistence works again, the re-ack removes the message.
	r.store = &passingStore{}
	if err := r.Ack(id); err != nil {
		t.Fatalf("Ack retry after recovery: want nil, got %v", err)
	}
	if _, ok := r.byID[id]; ok {
		t.Fatalf("after successful retry Ack, message should be removed from byID")
	}
	if contains(r.order, id) {
		t.Fatalf("after successful retry Ack, id should be removed from order")
	}
	if got := r.metrics.Snapshot().Acked; got != 1 {
		t.Fatalf("Acked counter = %d, want 1 after successful retry", got)
	}
}

func TestAckSuccessRemovesMessageAndPersists(t *testing.T) {
	r, id := newAckTestRelay()
	store := &passingStore{}
	r.store = store

	if err := r.Ack(id); err != nil {
		t.Fatalf("Ack: want nil, got %v", err)
	}
	if _, ok := r.byID[id]; ok {
		t.Fatalf("after Ack, message should be removed from byID")
	}
	if contains(r.order, id) {
		t.Fatalf("after Ack, id should be removed from order")
	}
	if got := r.metrics.Snapshot().Acked; got != 1 {
		t.Fatalf("Acked counter = %d, want 1", got)
	}
	// The snapshot written to disk should reflect the empty queue.
	if n := len(store.last.Messages); n != 0 {
		t.Fatalf("persisted snapshot has %d messages, want 0", n)
	}
}

func TestAckAlreadyAckedStillErrors(t *testing.T) {
	r, id := newAckTestRelay()
	r.store = &passingStore{}

	if err := r.Ack(id); err != nil {
		t.Fatalf("first Ack: want nil, got %v", err)
	}
	// After a successful ack the message is gone, so a second ack reports
	// not-found rather than already-acked. This guards the half-success path:
	// a persist failure leaves the message present, and re-acking after
	// recovery must succeed (not claim already-acked).
	if err := r.Ack(id); err == nil {
		t.Fatal("second Ack: want error, got nil")
	} else if !errors.Is(err, ErrNotFound) {
		t.Fatalf("second Ack: want ErrNotFound, got %v", err)
	}
}
