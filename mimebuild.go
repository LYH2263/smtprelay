package smtprelay

import (
	"fmt"
	"strings"
	"time"

	intmime "github.com/LYH2263/go-smtprelay/internal/mime"
)

// BuildMIME assembles RawBody from MimeParts / headers / subject.
// If RawBody is already set and no parts exist, it normalizes headers only.
func BuildMIME(env *Envelope) ([]byte, error) {
	if env == nil {
		return nil, ErrInvalidEnv
	}
	headers := map[string]string{}
	for k, v := range env.Headers {
		headers[k] = v
	}
	if env.Subject != "" {
		if _, ok := headers["Subject"]; !ok {
			headers["Subject"] = env.Subject
		}
	}
	if _, ok := headers["From"]; !ok {
		headers["From"] = env.MailFrom
	}
	if _, ok := headers["To"]; !ok {
		addrs := env.RecipientAddresses()
		headers["To"] = strings.Join(addrs, ", ")
	}
	if _, ok := headers["Date"]; !ok {
		headers["Date"] = env.CreatedAt.UTC().Format(time.RFC1123Z)
		if env.CreatedAt.IsZero() {
			headers["Date"] = time.Now().UTC().Format(time.RFC1123Z)
		}
	}
	if _, ok := headers["MIME-Version"]; !ok {
		headers["MIME-Version"] = "1.0"
	}

	if len(env.MimeParts) == 0 {
		body := env.RawBody
		if body == nil {
			body = []byte{}
		}
		if _, ok := headers["Content-Type"]; !ok {
			headers["Content-Type"] = "text/plain; charset=utf-8"
		}
		return intmime.Serialize(headers, body), nil
	}

	if len(env.MimeParts) == 1 && env.MimeParts[0].Disposition == "" {
		p := env.MimeParts[0]
		ct := p.ContentType
		if ct == "" {
			ct = "text/plain"
		}
		if p.Charset != "" && !strings.Contains(strings.ToLower(ct), "charset=") {
			ct = fmt.Sprintf("%s; charset=%s", ct, p.Charset)
		}
		headers["Content-Type"] = ct
		return intmime.Serialize(headers, p.Data), nil
	}

	boundary := intmime.NewBoundary()
	headers["Content-Type"] = "multipart/mixed; boundary=" + boundary
	var body []byte
	for _, p := range env.MimeParts {
		partHeaders := map[string]string{}
		ct := p.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}
		if p.Charset != "" {
			ct = fmt.Sprintf("%s; charset=%s", ct, p.Charset)
		}
		partHeaders["Content-Type"] = ct
		if p.Disposition != "" {
			disp := p.Disposition
			if p.Filename != "" {
				disp = fmt.Sprintf("%s; filename=%q", disp, p.Filename)
			}
			partHeaders["Content-Disposition"] = disp
		}
		if p.ContentID != "" {
			partHeaders["Content-ID"] = p.ContentID
		}
		partHeaders["Content-Transfer-Encoding"] = "8bit"
		body = append(body, []byte("--"+boundary+"\r\n")...)
		body = append(body, intmime.EncodeHeaders(partHeaders)...)
		body = append(body, []byte("\r\n")...)
		body = append(body, p.Data...)
		if len(p.Data) == 0 || (p.Data[len(p.Data)-1] != '\n') {
			body = append(body, []byte("\r\n")...)
		}
	}
	body = append(body, []byte("--"+boundary+"--\r\n")...)
	return intmime.Serialize(headers, body), nil
}

// EnsureMIME fills env.RawBody via BuildMIME when empty or parts present.
func EnsureMIME(env *Envelope) error {
	if env == nil {
		return ErrInvalidEnv
	}
	if len(env.RawBody) > 0 && len(env.MimeParts) == 0 {
		return nil
	}
	raw, err := BuildMIME(env)
	if err != nil {
		return err
	}
	env.RawBody = raw
	return nil
}
