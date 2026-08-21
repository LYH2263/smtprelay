package smtprelay_test

import (
	"path/filepath"
	"testing"

	smtprelay "github.com/LYH2263/go-smtprelay"
)

func TestBug01_SubmitRawBodySliceAlias(t *testing.T) {
	dir := t.TempDir()
	r, err := smtprelay.New(smtprelay.Options{PersistPath: filepath.Join(dir, "q.json")})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	body := []byte("outbound-body-keep")
	id, err := r.Submit(&smtprelay.Envelope{
		MailFrom:   "a@example.com",
		Recipients: []smtprelay.Recipient{{Address: "b@example.com"}},
		Subject:    "alias",
		RawBody:    body,
	})
	if err != nil {
		t.Fatal(err)
	}
	body[0] = 'X'
	got, err := r.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if string(got.RawBody) != "outbound-body-keep" {
		t.Fatalf("queued RawBody polluted by caller alias: %q", got.RawBody)
	}
}
