package smtprelay_test

import (
	"path/filepath"
	"testing"

	smtprelay "github.com/LYH2263/go-smtprelay"
)

func TestBug10_CloseFlushesBeforeClearQueue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "q.json")
	r, err := smtprelay.New(smtprelay.Options{PersistPath: path})
	if err != nil {
		t.Fatal(err)
	}
	id, err := r.Submit(&smtprelay.Envelope{
		MailFrom:   "a@example.com",
		Recipients: []smtprelay.Recipient{{Address: "b@example.com"}},
		RawBody:    []byte("flush-before-clear"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	r2, err := smtprelay.New(smtprelay.Options{PersistPath: path})
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Close()
	got, err := r2.Get(id)
	if err != nil {
		t.Fatalf("Close cleared queue before flush; message lost on reopen: %v", err)
	}
	if string(got.RawBody) != "flush-before-clear" {
		t.Fatalf("body=%q", got.RawBody)
	}
}
