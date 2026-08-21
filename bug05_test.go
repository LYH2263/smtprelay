package smtprelay_test

import (
	"context"
	"errors"
	"testing"

	smtprelay "github.com/LYH2263/go-smtprelay"
	"github.com/LYH2263/go-smtprelay/internal/smtpdial"
)

func TestBug05_Permanent5xxWrapsErrPermanent(t *testing.T) {
	dialer := &smtpdial.MemoryDialer{Code: 550, Response: "user unknown", Err: errors.New("rcpt refused")}
	r, err := smtprelay.New(smtprelay.Options{
		Dialer:      dialer,
		DirectHost:  "mx.example.com",
		MaxAttempts: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	_, err = r.Submit(&smtprelay.Envelope{
		MailFrom:   "a@example.com",
		Recipients: []smtprelay.Recipient{{Address: "b@example.com"}},
		RawBody:    []byte("bounce-me"),
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := r.DeliverOnce(context.Background())
	if err == nil {
		t.Fatal("expected permanent delivery error")
	}
	if !errors.Is(err, smtprelay.ErrPermanent) {
		t.Fatalf("want errors.Is ErrPermanent, got %v", err)
	}
	if smtprelay.ClassifyBounce(res) != smtprelay.BouncePermanent {
		t.Fatalf("bounce class=%s want permanent (res=%+v)", smtprelay.ClassifyBounce(res), res)
	}
	if len(r.ListBounces(10)) != 1 {
		t.Fatalf("expected 1 bounce record, got %d", len(r.ListBounces(10)))
	}
}
