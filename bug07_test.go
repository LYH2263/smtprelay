package smtprelay_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	smtprelay "github.com/LYH2263/go-smtprelay"
)

func TestBug07_SubmitContextHonorsCancel(t *testing.T) {
	dir := t.TempDir()
	r, err := smtprelay.New(smtprelay.Options{PersistPath: filepath.Join(dir, "q.json")})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = r.SubmitContext(ctx, &smtprelay.Envelope{
		MailFrom:   "a@example.com",
		Recipients: []smtprelay.Recipient{{Address: "b@example.com"}},
		RawBody:    []byte("should-not-enqueue"),
	})
	if err == nil {
		t.Fatal("expected context cancel error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if st := r.Stats(); st.Total != 0 {
		t.Fatalf("canceled Submit still enqueued: %+v", st)
	}
}
