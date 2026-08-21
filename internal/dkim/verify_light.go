package dkim

import (
        "strings"
)

// ParseSignatureTags splits a DKIM-Signature value into tags.
func ParseSignatureTags(sig string) map[string]string {
        out := map[string]string{}
        for _, part := range strings.Split(sig, ";") {
                part = strings.TrimSpace(part)
                if part == "" {
                        continue
                }
                i := strings.IndexByte(part, '=')
                if i <= 0 {
                        continue
                }
                k := strings.TrimSpace(part[:i])
                v := strings.TrimSpace(part[i+1:])
                out[k] = v
        }
        return out
}

// HasRequiredTags checks v/a/d/s/bh/h/b presence.
func HasRequiredTags(tags map[string]string) bool {
        for _, k := range []string{"v", "a", "d", "s", "bh", "h", "b"} {
                if tags[k] == "" {
                        return false
                }
        }
        return true
}
