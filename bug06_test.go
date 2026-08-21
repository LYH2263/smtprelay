package smtprelay_test

import (
	"os"
	"path/filepath"
	"testing"

	smtprelay "github.com/LYH2263/go-smtprelay"
)

func TestBug06_AckPersistFailureKeepsMessage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "q.json")
	r, err := smtprelay.New(smtprelay.Options{PersistPath: path})
	if err != nil {
		t.Fatal(err)
	}
	id, err := r.Submit(&smtprelay.Envelope{
		MailFrom:   "a@example.com",
		Recipients: []smtprelay.Recipient{{Address: "b@example.com"}},
		RawBody:    []byte("must-survive-ack-fail"),
	})
	if err != nil {
		t.Fatal(err)
	}
	// 把快照路径换成目录，迫使后续 Ack 的 Rename/写盘失败
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	err = r.Ack(id)
	if err == nil {
		t.Fatal("expected persist error from Ack")
	}
	// 失败后消息必须仍在内存队列
	if _, err := r.Get(id); err != nil {
		t.Fatalf("message lost from memory after failed Ack: %v", err)
	}
	// 恢复可写路径后再 Close，确认能把仍在队列的信刷盘
	if err := os.RemoveAll(path); err != nil {
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
	if _, err := r2.Get(id); err != nil {
		t.Fatalf("message missing after restart (Ack persist failure dropped mail): %v", err)
	}
}
