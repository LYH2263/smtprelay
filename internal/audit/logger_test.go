package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRotateReleasesHandle verifies the Windows-safe order (Close before
// Rename): Rotate must succeed and release the handle on the rotated
// archive so an external process can delete it. The previous order
// (Rename before Close) left delivery.log handle-bound on Windows, so
// os.Rename failed and Close never ran — the archive stayed locked and
// logging wedged.
func TestRotateReleasesHandle(t *testing.T) {
	dir := t.TempDir()
	l, err := OpenDir(dir)
	if err != nil {
		t.Fatalf("OpenDir: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })

	if err := l.LogDelivery("msg-1", "mx.example.com", 250, "OK", nil); err != nil {
		t.Fatalf("LogDelivery: %v", err)
	}

	// Rename must succeed: on the buggy order it failed because the handle
	// was still open, and the function returned early without closing.
	if err := l.Rotate(); err != nil {
		t.Fatalf("Rotate: %v", err)
	}

	// A fresh delivery.log is reopened and the stamped archive is on disk.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var haveLog, haveArchive bool
	var archive string
	for _, e := range entries {
		name := e.Name()
		if name == "delivery.log" {
			haveLog = true
		}
		if strings.HasPrefix(name, "delivery-") && strings.HasSuffix(name, ".log") {
			haveArchive = true
			archive = name
		}
	}
	if !haveArchive {
		t.Fatal("rotated archive not created")
	}

	// The archive (the old delivery.log) must no longer be held open by us —
	// close-before-rename guarantees the handle is dropped before the rename,
	// and we never reopen the archive. An external process can delete it.
	if err := os.Remove(filepath.Join(dir, archive)); err != nil {
		t.Fatalf("archive %s still locked after Rotate: %v", archive, err)
	}

	// The logger must not be wedged: it reopened delivery.log and can log
	// again. (We must NOT assert delivery.log is deletable here — it is
	// legitimately held open for continued writing.)
	if err := l.LogDelivery("msg-2", "mx.example.com", 250, "OK", nil); err != nil {
		t.Fatalf("LogDelivery after Rotate: %v", err)
	}
	if !haveLog {
		t.Fatal("delivery.log not reopened after Rotate")
	}
}

// TestCloseDoesNotRotate ensures Close shuts down without producing a rotated
// archive — the previous Close path called Rotate() before Close(), which
// raced the rename against the final close on Windows.
func TestCloseDoesNotRotate(t *testing.T) {
	dir := t.TempDir()
	l, err := OpenDir(dir)
	if err != nil {
		t.Fatalf("OpenDir: %v", err)
	}
	if err := l.LogDelivery("msg-2", "mx.example.com", 250, "OK", nil); err != nil {
		t.Fatalf("LogDelivery: %v", err)
	}

	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "delivery-") {
			t.Fatalf("Close produced a rotated archive %q — Close must not Rotate", e.Name())
		}
	}
}
