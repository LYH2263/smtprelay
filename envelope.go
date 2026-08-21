package smtprelay

import (
	"strings"
	"time"

	"github.com/LYH2263/go-smtprelay/internal/clone"
	"github.com/LYH2263/go-smtprelay/internal/validate"
)

// Recipient is one RCPT TO target.
type Recipient struct {
	Address string
	Name    string
}

// Envelope is the SMTP transaction payload queued for delivery.
type Envelope struct {
	ID          string
	MailFrom    string
	Recipients  []Recipient
	Subject     string
	Headers     map[string]string
	RawBody     []byte
	MimeParts   []MimePart
	CreatedAt   time.Time
	Attempts    int
	NextAttempt time.Time
	State       MessageState
	LastError   string
	DKIMSigned  bool
}

// MimePart is a MIME body part before assembly.
type MimePart struct {
	ContentType string
	Charset     string
	Disposition string
	Filename    string
	ContentID   string
	Data        []byte
}

// Clone returns a deep copy safe for callers to mutate.
func (e *Envelope) Clone() *Envelope {
	if e == nil {
		return nil
	}
	out := *e
	out.Recipients = cloneRecipients(e.Recipients)
	out.Headers = clone.HeaderMap(e.Headers)
	out.RawBody = clone.Bytes(e.RawBody)
	out.MimeParts = cloneMimeParts(e.MimeParts)
	return &out
}

func cloneRecipients(in []Recipient) []Recipient {
	if in == nil {
		return nil
	}
	out := make([]Recipient, len(in))
	copy(out, in)
	return out
}

func cloneMimeParts(in []MimePart) []MimePart {
	if in == nil {
		return nil
	}
	out := make([]MimePart, len(in))
	for i, p := range in {
		out[i] = p
		out[i].Data = clone.Bytes(p.Data)
	}
	return out
}

// Validate checks MAIL FROM / RCPT TO / body presence.
func (e *Envelope) Validate() error {
	if e == nil {
		return ErrInvalidEnv
	}
	if err := validate.Mailbox(e.MailFrom); err != nil {
		return err
	}
	if len(e.Recipients) == 0 {
		return ErrInvalidEnv
	}
	for _, r := range e.Recipients {
		if err := validate.Mailbox(r.Address); err != nil {
			return err
		}
	}
	if len(e.RawBody) == 0 && len(e.MimeParts) == 0 {
		return ErrEmptyBody
	}
	return nil
}

// RecipientAddresses returns bare addresses.
func (e *Envelope) RecipientAddresses() []string {
	out := make([]string, 0, len(e.Recipients))
	for _, r := range e.Recipients {
		out = append(out, strings.TrimSpace(r.Address))
	}
	return out
}
