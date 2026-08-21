package validate

import (
        "errors"
        "strings"
)

var ErrBadHeader = errors.New("validate: bad header name")

// HeaderName checks RFC 5322 token-ish header names.
func HeaderName(name string) error {
        name = strings.TrimSpace(name)
        if name == "" {
                return ErrBadHeader
        }
        for _, r := range name {
                if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
                        continue
                }
                return ErrBadHeader
        }
        return nil
}

// NormalizeHeaderKey canonicalizes common mail headers.
func NormalizeHeaderKey(k string) string {
        parts := strings.Split(k, "-")
        for i, p := range parts {
                if p == "" {
                        continue
                }
                low := strings.ToLower(p)
                parts[i] = strings.ToUpper(low[:1]) + low[1:]
        }
        return strings.Join(parts, "-")
}
