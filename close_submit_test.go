package smtprelay

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// validEnv returns a minimal envelope that passes Validate.
func validEnv() *Envelope {
	return &Envelope{
		MailFrom:   "a@b.com",
		Recipients: []Recipient{{Address: "c@d.com"}},
		RawBody:    []byte("Subject: x\r\n\r\nbody"),
	}
}

// Submit after Close must return ErrClosed, not panic on a nil map.
// Regression: SubmitContext used to write r.byID[id] without a closed guard,
// and Close nils r.byID, yielding "assignment to entry in nil map".
func TestSubmitAfterCloseReturnsErrClosed(t *testing.T) {
	r, err := New(Options{PersistPath: tmpPersistPath(t)})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Must not panic; must return ErrClosed.
	id, err := r.Submit(validEnv())
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("Submit after Close: want ErrClosed, got id=%q err=%v", id, err)
	}
}

// SubmitContext after Close must likewise return ErrClosed, not panic.
func TestSubmitContextAfterCloseReturnsErrClosed(t *testing.T) {
	r, err := New(Options{PersistPath: tmpPersistPath(t)})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	id, err := r.SubmitContext(context.Background(), validEnv())
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("SubmitContext after Close: want ErrClosed, got id=%q err=%v", id, err)
	}
}

// Closed state must be sticky; a second Close is a no-op returning nil.
func TestDoubleCloseIsNoop(t *testing.T) {
	r, err := New(Options{PersistPath: tmpPersistPath(t)})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("second Close: want nil, got %v", err)
	}
	if !r.Closed() {
		t.Fatalf("Closed() = false, want true")
	}
}

// tmpPersistPath returns a per-test snapshot file path, cleaned up on exit.
func tmpPersistPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "queue.json")
}

// Ensure no stray fixture files leak from earlier ad-hoc repros.
func TestMain(m *testing.M) {
	_ = os.Remove("nonexistent_repro.json")
	os.Exit(m.Run())
}
