package smtprelay

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newTestRelay builds a Relay whose persistence points at path, with a frozen
// clock so NextAttempt ordering is deterministic.
func newTestRelay(t *testing.T, path string) *Relay {
	t.Helper()
	fixed := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	r, err := New(Options{
		PersistPath: path,
		Now:         func() time.Time { return fixed },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return r
}

// envFor returns a minimal valid envelope addressed to the given recipient.
func envFor(mailFrom, rcpt string) *Envelope {
	return &Envelope{
		MailFrom:   mailFrom,
		Recipients: []Recipient{{Address: rcpt}},
		RawBody:    []byte("hello"),
	}
}

// TestClosePersistsUndeliveredQueue is the regression for the bug where Close
// cleared byID/order before flushing, writing an empty snapshot and losing
// every undelivered message on reopen with the same PersistPath.
func TestClosePersistsUndeliveredQueue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "queue.json")

	// First instance: enqueue two messages, then close WITHOUT delivering.
	r := newTestRelay(t, path)
	t.Cleanup(func() { _ = r.Close() })

	id1, err := r.Submit(envFor("alice@example.com", "bob@example.com"))
	if err != nil {
		t.Fatalf("Submit 1: %v", err)
	}
	id2, err := r.Submit(envFor("alice@example.com", "carol@example.com"))
	if err != nil {
		t.Fatalf("Submit 2: %v", err)
	}

	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Reopen at the same path: the queue must still contain both messages.
	r2 := newTestRelay(t, path)
	defer r2.Close()

	wantIDs := map[string]bool{id1: true, id2: true}
	got := map[string]bool{}
	for _, env := range r2.ListPending() {
		got[env.ID] = true
	}
	for id := range wantIDs {
		if !got[id] {
			t.Errorf("after reopen, message %q missing from queue; got=%v", id, got)
		}
	}
	if len(got) != len(wantIDs) {
		t.Errorf("after reopen, queue len = %d, want %d (got %v)", len(got), len(wantIDs), got)
	}

	// The on-disk file must not be an empty snapshot.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat snapshot: %v", err)
	}
	if info.Size() < 2 {
		t.Fatalf("snapshot is empty after Close; size=%d", info.Size())
	}
}
