package audit_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LYH2263/go-smtprelay/internal/audit"
)

func TestBug09_AuditRotateClosesBeforeRename(t *testing.T) {
	dir := t.TempDir()
	lg, err := audit.OpenDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer lg.Close()
	if err := lg.LogDelivery("msg-1", "mx.example.com", 250, "OK", nil); err != nil {
		t.Fatal(err)
	}
	if err := lg.Rotate(); err != nil {
		t.Fatalf("Rotate failed (Close before Rename required on Windows): %v", err)
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var stamped bool
	for _, e := range ents {
		name := e.Name()
		if name != "delivery.log" && filepath.Ext(name) == ".log" {
			stamped = true
		}
	}
	if !stamped {
		t.Fatal("expected rotated delivery-*.log after Rotate")
	}
	if err := lg.LogDelivery("msg-2", "mx.example.com", 250, "OK", nil); err != nil {
		t.Fatal(err)
	}
}
