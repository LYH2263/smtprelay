package persist

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Recipient persisted form.
type Recipient struct {
	Address string `json:"address"`
	Name    string `json:"name,omitempty"`
}

// MimePart persisted form.
type MimePart struct {
	ContentType string `json:"content_type"`
	Charset     string `json:"charset,omitempty"`
	Disposition string `json:"disposition,omitempty"`
	Filename    string `json:"filename,omitempty"`
	ContentID   string `json:"content_id,omitempty"`
	Data        []byte `json:"data,omitempty"`
}

// Message is one queued envelope snapshot.
type Message struct {
	ID          string            `json:"id"`
	MailFrom    string            `json:"mail_from"`
	Recipients  []Recipient       `json:"recipients"`
	Subject     string            `json:"subject,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	RawBody     []byte            `json:"raw_body,omitempty"`
	MimeParts   []MimePart        `json:"mime_parts,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	Attempts    int               `json:"attempts"`
	NextAttempt time.Time         `json:"next_attempt"`
	State       string            `json:"state"`
	LastError   string            `json:"last_error,omitempty"`
	DKIMSigned  bool              `json:"dkim_signed,omitempty"`
}

// Snapshot is the on-disk queue file.
type Snapshot struct {
	SavedAt  time.Time `json:"saved_at"`
	Messages []Message `json:"messages"`
}

// Store is a JSON file-backed queue snapshot.
type Store struct {
	path string
	mu   sync.Mutex
}

// Open prepares a store at path (file may not exist yet).
func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("persist: empty path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return &Store{path: path}, nil
}

// Load reads the snapshot; missing file → empty.
func (s *Store) Load() (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return Snapshot{Messages: nil}, nil
		}
		return Snapshot{}, err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return Snapshot{}, err
	}
	return snap, nil
}

// Save writes atomically via temp + rename.
func (s *Store) Save(ctx context.Context, snap Snapshot) error {

	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Close is a no-op placeholder for lifecycle symmetry.
func (s *Store) Close() error { return nil }

// Path returns the file path.
func (s *Store) Path() string { return s.path }
