package validate

import (
        "errors"
        "strings"
        "unicode"
)

var (
        ErrEmptyMailbox  = errors.New("validate: empty mailbox")
        ErrBadMailbox    = errors.New("validate: malformed mailbox")
        ErrMailboxLength = errors.New("validate: mailbox too long")
)

// Mailbox checks a minimal addr-spec (local@domain).
func Mailbox(addr string) error {
        addr = strings.TrimSpace(addr)
        if addr == "" {
                return ErrEmptyMailbox
        }
        if len(addr) > 254 {
                return ErrMailboxLength
        }
        // Allow "Name <local@domain>"
        if i := strings.LastIndex(addr, "<"); i >= 0 {
                j := strings.LastIndex(addr, ">")
                if j <= i {
                        return ErrBadMailbox
                }
                addr = addr[i+1 : j]
        }
        at := strings.LastIndex(addr, "@")
        if at <= 0 || at == len(addr)-1 {
                return ErrBadMailbox
        }
        local, domain := addr[:at], addr[at+1:]
        if !tokenOK(local, true) || !domainOK(domain) {
                return ErrBadMailbox
        }
        return nil
}

func tokenOK(s string, allowDot bool) bool {
        if s == "" {
                return false
        }
        for i, r := range s {
                if r == '.' {
                        if !allowDot || i == 0 || i == len(s)-1 {
                                return false
                        }
                        continue
                }
                if unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("!#$%&'*+-/=?^_`{|}~", r) {
                        continue
                }
                return false
        }
        return true
}

func domainOK(s string) bool {
        if s == "" || strings.HasPrefix(s, ".") || strings.HasSuffix(s, ".") {
                return false
        }
        for _, lab := range strings.Split(s, ".") {
                if lab == "" || len(lab) > 63 {
                        return false
                }
                for i, r := range lab {
                        if unicode.IsLetter(r) || unicode.IsDigit(r) {
                                continue
                        }
                        if r == '-' && i > 0 && i < len(lab)-1 {
                                continue
                        }
                        return false
                }
        }
        return true
}

// Domain extracts domain from mailbox.
func Domain(addr string) (string, error) {
        if err := Mailbox(addr); err != nil {
                return "", err
        }
        addr = strings.TrimSpace(addr)
        if i := strings.LastIndex(addr, "<"); i >= 0 {
                j := strings.LastIndex(addr, ">")
                addr = addr[i+1 : j]
        }
        at := strings.LastIndex(addr, "@")
        return addr[at+1:], nil
}
