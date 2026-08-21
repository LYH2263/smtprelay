package smtprelay_test

import (
	"context"
	"testing"

	smtprelay "github.com/LYH2263/go-smtprelay"
	"github.com/LYH2263/go-smtprelay/internal/smtpdial"
)

func TestBug04_NilHeadersDeliverNoInFlightStuck(t *testing.T) {
	dialer := &smtpdial.MemoryDialer{Code: 250, Response: "OK"}
	r, err := smtprelay.New(smtprelay.Options{
		Dialer:     dialer,
		DirectHost: "mx.example.com",
		DirectPort: 25,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	id, err := r.Submit(&smtprelay.Envelope{
		MailFrom:   "a@example.com",
		Recipients: []smtprelay.Recipient{{Address: "b@example.com"}},
		RawBody:    []byte("Subject: nil-headers\r\n\r\nbody"),
		Headers:    nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("DeliverOnce panicked on nil Headers after enqueue: %v", rec)
		}
	}()
	_, err = r.DeliverOnce(context.Background())
	if err != nil {
		t.Fatalf("DeliverOnce err=%v", err)
	}
	got, err := r.Get(id)
	if err == nil && got.State == smtprelay.StateInFlight {
		t.Fatalf("message stuck InFlight after nil-Headers path: %+v", got)
	}
	for _, e := range r.ListPending() {
		if e.ID == id && e.State == smtprelay.StateInFlight {
			t.Fatalf("pending list still shows InFlight for %s", id)
		}
	}
}
