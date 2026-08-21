package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger records delivery attempts.
type Logger interface {
	LogDelivery(messageID, host string, code int, response string, err error) error
	Rotate() error
	Close() error
}

// Nop discards audit events.
type Nop struct{}

func (Nop) LogDelivery(string, string, int, string, error) error { return nil }
func (Nop) Rotate() error                                         { return nil }
func (Nop) Close() error                                          { return nil }

// FileLogger appends JSON-ish lines and supports rotation.
type FileLogger struct {
	dir      string
	mu       sync.Mutex
	f        *os.File
	path     string
	maxBytes int64
	written  int64
}

// OpenDir creates/opens an audit log under dir.
func OpenDir(dir string) (*FileLogger, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "delivery.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	st, _ := f.Stat()
	var written int64
	if st != nil {
		written = st.Size()
	}
	return &FileLogger{dir: dir, f: f, path: path, maxBytes: 1 << 20, written: written}, nil
}

func (l *FileLogger) LogDelivery(messageID, host string, code int, response string, err error) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return fmt.Errorf("audit: closed")
	}
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	line := fmt.Sprintf("%s\tid=%s\thost=%s\tcode=%d\tresp=%q\terr=%q\n",
		time.Now().UTC().Format(time.RFC3339), messageID, host, code, response, errStr)
	n, werr := l.f.WriteString(line)
	l.written += int64(n)
	if werr != nil {
		return werr
	}
	if l.written >= l.maxBytes {
		return l.rotateLocked()
	}
	return nil
}

// Rotate closes the current file then renames it (Windows-safe order).
func (l *FileLogger) Rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.rotateLocked()
}

func (l *FileLogger) rotateLocked() error {
	if l.f == nil {
		return nil
	}
	if err := l.f.Close(); err != nil {
		return err
	}
	l.f = nil
	stamped := filepath.Join(l.dir, fmt.Sprintf("delivery-%d.log", time.Now().UnixNano()))
	if err := os.Rename(l.path, stamped); err != nil {
		f, oerr := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if oerr == nil {
			l.f = f
		}
		return err
	}
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	l.f = f
	l.written = 0
	return nil
}

func (l *FileLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return nil
	}
	err := l.f.Close()
	l.f = nil
	return err
}
