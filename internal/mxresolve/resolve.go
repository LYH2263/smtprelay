package mxresolve

import (
        "errors"
        "net"
        "sort"
        "strings"
)

// Resolver looks up MX hosts for a domain.
type Resolver interface {
        LookupMX(domain string) ([]string, error)
}

// Static returns the domain itself as the only "MX" (lab default).
type Static struct{}

func (Static) LookupMX(domain string) ([]string, error) {
        domain = strings.TrimSpace(strings.ToLower(domain))
        if domain == "" {
                return nil, errors.New("mxresolve: empty domain")
        }
        return []string{domain}, nil
}

// DNS uses net.LookupMX.
type DNS struct{}

func (DNS) LookupMX(domain string) ([]string, error) {
        domain = strings.TrimSpace(domain)
        if domain == "" {
                return nil, errors.New("mxresolve: empty domain")
        }
        records, err := net.LookupMX(domain)
        if err != nil {
                return nil, err
        }
        sort.Slice(records, func(i, j int) bool {
                return records[i].Pref < records[j].Pref
        })
        out := make([]string, 0, len(records))
        for _, r := range records {
                host := strings.TrimSuffix(r.Host, ".")
                if host != "" {
                        out = append(out, host)
                }
        }
        if len(out) == 0 {
                return []string{domain}, nil
        }
        return out, nil
}

// MapResolver is a fixed domain→hosts table for tests.
type MapResolver map[string][]string

func (m MapResolver) LookupMX(domain string) ([]string, error) {
        domain = strings.ToLower(strings.TrimSpace(domain))
        if hs, ok := m[domain]; ok {
                out := append([]string(nil), hs...)
                return out, nil
        }
        return nil, errors.New("mxresolve: unknown domain")
}
