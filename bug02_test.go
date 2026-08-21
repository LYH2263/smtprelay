package smtprelay_test

import (
	"testing"

	smtprelay "github.com/LYH2263/go-smtprelay"
)

func TestBug02_ListPendingRecipientsAlias(t *testing.T) {
	r, err := smtprelay.New(smtprelay.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	_, err = r.Submit(&smtprelay.Envelope{
		MailFrom:   "a@example.com",
		Recipients: []smtprelay.Recipient{{Address: "keep@example.com", Name: "K"}},
		RawBody:    []byte("x"),
	})
	if err != nil {
		t.Fatal(err)
	}
	list := r.ListPending()
	if len(list) != 1 {
		t.Fatalf("len=%d", len(list))
	}
	list[0].Recipients[0].Address = "mutated@evil.test"
	again := r.ListPending()
	if again[0].Recipients[0].Address != "keep@example.com" {
		t.Fatalf("ListPending Recipients aliased into internal table: %v", again[0].Recipients)
	}
}
