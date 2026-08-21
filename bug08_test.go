package smtprelay_test

import (
	"context"
	"errors"
	"testing"
	"time"

	smtprelay "github.com/LYH2263/go-smtprelay"
	"github.com/LYH2263/go-smtprelay/internal/smtpdial"
)

func TestBug08_DialAndSendHonorsContext(t *testing.T) {
	dialer := &smtpdial.MemoryDialer{Delay: 300 * time.Millisecond, Code: 250, Response: "OK"}
	r, err := smtprelay.New(smtprelay.Options{Dialer: dialer})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = r.DialAndSend(ctx, dialer, "mx.example.com", 25, &smtprelay.Envelope{
		MailFrom:   "a@example.com",
		Recipients: []smtprelay.Recipient{{Address: "b@example.com"}},
		RawBody:    []byte("slow"),
	})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected ctx deadline/cancel")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("want ctx error, got %v", err)
	}
	if elapsed > 200*time.Millisecond {
		t.Fatalf("DialAndSend ignored cancel, blocked %v", elapsed)
	}
}
