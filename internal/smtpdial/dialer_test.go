package smtpdial

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestMemoryDialerCancelReturnsFast ensures a cancelled ctx short-circuits the
// Delay instead of blocking for its full duration.
func TestMemoryDialerCancelReturnsFast(t *testing.T) {
	m := &MemoryDialer{Delay: 2 * time.Second, Code: 250}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, _, err := m.Send(ctx, Request{Host: "x", Port: 25})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected ctx.Err() to propagate, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if elapsed > time.Second {
		t.Fatalf("Send ignored ctx and waited the full Delay (elapsed=%v)", elapsed)
	}
}
