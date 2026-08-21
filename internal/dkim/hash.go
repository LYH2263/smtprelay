        package dkim

        import (
        	"crypto/sha256"
        	"encoding/base64"
        	"strings"
        )

        // BodyHash computes the DKIM body hash under relaxed canonicalization.
        func BodyHash(body []byte, canon string) string {
        	c := strings.ToLower(canon)
        	if strings.Contains(c, "simple") && !strings.Contains(c, "relaxed") {
        		b := normalizeSimpleBody(body)
        		sum := sha256.Sum256(b)
        		return base64.StdEncoding.EncodeToString(sum[:])
        	}
        	b := relaxBody(body)
        	sum := sha256.Sum256(b)
        	return base64.StdEncoding.EncodeToString(sum[:])
        }

        func relaxBody(body []byte) []byte {
        	s := string(body)
        	s = strings.ReplaceAll(s, "\r\n", "\n")
        	s = strings.ReplaceAll(s, "\r", "\n")
        	lines := strings.Split(s, "\n")
        		for i, line := range lines {
		line = strings.TrimRight(line, " \t")
		line = collapseWS(line)
		lines[i] = line
	}

        	for len(lines) > 0 && lines[len(lines)-1] == "" {
        		lines = lines[:len(lines)-1]
        	}
        	if len(lines) == 0 {
        		return []byte("\r\n")
        	}
        	return []byte(strings.Join(lines, "\r\n") + "\r\n")
        }

        func normalizeSimpleBody(body []byte) []byte {
        	s := string(body)
        	s = strings.ReplaceAll(s, "\r\n", "\n")
        	s = strings.ReplaceAll(s, "\r", "\n")
        	for strings.HasSuffix(s, "\n") {
        		s = strings.TrimSuffix(s, "\n")
        	}
        	if s == "" {
        		return []byte("\r\n")
        	}
        	return []byte(strings.ReplaceAll(s, "\n", "\r\n") + "\r\n")
        }

        func collapseWS(s string) string {
        	var b strings.Builder
        	prevSpace := false
        	for _, r := range s {
        		if r == ' ' || r == '\t' {
        			if prevSpace {
        				continue
        			}
        			b.WriteByte(' ')
        			prevSpace = true
        			continue
        		}
        		prevSpace = false
        		b.WriteRune(r)
        	}
        	return b.String()
        }

        // HeaderHashInput builds the DKIM signing string for listed headers.
        func HeaderHashInput(headers map[string]string, names []string) string {
        	var b strings.Builder
        	for _, name := range names {
        		val := findHeader(headers, name)
        		b.WriteString(strings.ToLower(name))
        		b.WriteString(":")
        		b.WriteString(relaxHeaderValue(val))
        		b.WriteString("\r\n")
        	}
        	return b.String()
        }

        func findHeader(headers map[string]string, name string) string {
        	for k, v := range headers {
        		if strings.EqualFold(k, name) {
        			return v
        		}
        	}
        	return ""
        }

        func relaxHeaderValue(v string) string {
        	v = strings.TrimSpace(v)
        	v = collapseWS(v)
        	return v
        }
