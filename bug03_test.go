package smtprelay_test

import (
	"errors"
	"path/filepath"
	"testing"

	smtprelay "github.com/LYH2263/go-smtprelay"
)

func TestBug03_SubmitAfterCloseNoPanic(t *testing.T) {
	dir := t.TempDir()
	r, err := smtprelay.New(smtprelay.Options{PersistPath: filepath.Join(dir, "q.json")})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("Submit after Close panicked: %v", rec)
		}
	}()
	_, err = r.Submit(&smtprelay.Envelope{
		MailFrom:   "a@example.com",
		Recipients: []smtprelay.Recipient{{Address: "b@example.com"}},
		RawBody:    []byte("after-close"),
	})
	if !errors.Is(err, smtprelay.ErrClosed) {
		t.Fatalf("want ErrClosed, got %v", err)
	}
}
