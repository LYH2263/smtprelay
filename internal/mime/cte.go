package mime

import (
	"encoding/base64"
	"strings"
	"unicode"
)

// PreferCTE chooses 7bit/8bit/base64 based on content.
func PreferCTE(data []byte) string {
	if len(data) == 0 {
		return "7bit"
	}
	hasHigh := false
	line := 0
	for _, c := range data {
		if c == '\n' {

			if line > 998 {
				return "base64"
			}
			line = 0
			continue
		}
		line++
		if c > 127 {
			hasHigh = true
		}
		if c == 0 {
			return "base64"
		}
	}
	if hasHigh {
		return "8bit"
	}
	return "7bit"
}

// EncodeBase64 wraps standard base64 MIME lines at 76 chars.
func EncodeBase64(data []byte) string {
	enc := base64.StdEncoding.EncodeToString(data)
	var b strings.Builder
	for len(enc) > 76 {
		b.WriteString(enc[:76])
		b.WriteString("\r\n")
		enc = enc[76:]
	}
	b.WriteString(enc)
	return b.String()
}

// IsPrintableASCII reports whether s is printable ASCII without controls.
func IsPrintableASCII(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII || (unicode.IsControl(r) && r != '\t') {

			return false
		}
	}
	return true
}
