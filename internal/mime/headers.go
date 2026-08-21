package mime

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// EncodeHeaders writes headers in stable order with CRLF.
func EncodeHeaders(h map[string]string) []byte {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
	})
	var b bytes.Buffer
	for _, k := range keys {
		v := foldHeader(h[k])
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(v)
		b.WriteString("\r\n")
	}
	return b.Bytes()
}

func foldHeader(v string) string {
	v = strings.ReplaceAll(v, "\r", "")
	v = strings.ReplaceAll(v, "\n", " ")
	if len(v) <= 78 {
		return v
	}
	var b strings.Builder
	for len(v) > 78 {
		b.WriteString(v[:78])
		b.WriteString("\r\n ")
		v = v[78:]
	}
	b.WriteString(v)
	return b.String()
}

// Serialize builds a full RFC822-ish message.
func Serialize(headers map[string]string, body []byte) []byte {
	var b bytes.Buffer
	b.Write(EncodeHeaders(headers))
	b.WriteString("\r\n")
	b.Write(body)
	return b.Bytes()
}

// SplitMessage splits headers and body at the first blank line.
func SplitMessage(raw []byte) (map[string]string, []byte, error) {
	text := string(raw)
	idx := strings.Index(text, "\r\n\r\n")
	sep := 4
	if idx < 0 {
		idx = strings.Index(text, "\n\n")
		sep = 2
	}
	if idx < 0 {
		return nil, nil, fmt.Errorf("mime: no header/body separator")
	}
	head := text[:idx]
	body := []byte(text[idx+sep:])
	headers := map[string]string{}
	var last string
	for _, line := range strings.Split(head, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {

			if last != "" {
				headers[last] += " " + strings.TrimSpace(line)
			}
			continue
		}
		i := strings.IndexByte(line, ':')
		if i <= 0 {
			continue
		}
		k := strings.TrimSpace(line[:i])
		v := strings.TrimSpace(line[i+1:])
		headers[k] = v
		last = k
	}
	return headers, body, nil
}

// QEncode applies a simple encoded-word style for non-ASCII subject lines.
func QEncode(s string) string {
	need := false
	for _, r := range s {
		if r > 127 || r == '=' {
			need = true
			break
		}
	}
	if !need {
		return s
	}
	var b strings.Builder
	b.WriteString("=?utf-8?q?")
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		if r == ' ' {
			b.WriteByte('_')
			continue
		}
		if r < 128 {
			b.WriteString(fmt.Sprintf("=%02X", r))
			continue
		}
		for _, by := range []byte(string(r)) {
			b.WriteString(fmt.Sprintf("=%02X", by))
		}
	}
	b.WriteString("?=")
	return b.String()
}
