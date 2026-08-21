package smtprelay

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LYH2263/go-smtprelay/internal/smtpdial"
)

// slowDialer blocks forever on Send unless ctx is honoured.
type slowDialer struct{}

func (slowDialer) Send(ctx context.Context, req smtpdial.Request) (int, string, error) {
	<-ctx.Done()
	return 0, "", ctx.Err()
}

// TestDialAndSendCancelDuringHandshake verifies the 50ms handshake wait
// returns promptly when the caller cancels mid-wait.
func TestDialAndSendCancelDuringHandshake(t *testing.T) {
	r := &Relay{opts: Options{}.withDefaults()}
	env := &Envelope{MailFrom: "a@b.com", RawBody: []byte("x")}
	env.Recipients = []Recipient{{Address: "c@d.com"}}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := r.DialAndSend(ctx, slowDialer{}, "127.0.0.1", 25, env)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if elapsed > time.Second {
		t.Fatalf("handshake wait did not honor ctx (elapsed=%v)", elapsed)
	}
}

// TestDialAndSendCancelDuringSend verifies the dialer's ctx is the caller's
// ctx, so cancellation inside Send propagates and returns promptly.
func TestDialAndSendCancelDuringSend(t *testing.T) {
	r := &Relay{opts: Options{}.withDefaults()}
	env := &Envelope{MailFrom: "a@b.com", RawBody: []byte("x")}
	env.Recipients = []Recipient{{Address: "c@d.com"}}

	// Pre-cancel so the handshake wait (50ms) completes normally but the
	// dialer sees an already-done ctx and returns immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := r.DialAndSend(ctx, slowDialer{}, "127.0.0.1", 25, env)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if elapsed > time.Second {
		t.Fatalf("dialer.Send did not honor ctx (elapsed=%v)", elapsed)
	}
}
