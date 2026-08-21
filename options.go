package smtprelay

import (
        "time"

        "github.com/LYH2263/go-smtprelay/internal/audit"
        "github.com/LYH2263/go-smtprelay/internal/mxresolve"
        "github.com/LYH2263/go-smtprelay/internal/smtpdial"
)

// Options configures a Relay.
type Options struct {
        // MaxQueue caps pending+deferred+inflight messages.
        MaxQueue int
        // PersistPath is the JSON queue snapshot path (optional).
        PersistPath string
        // AuditDir is where delivery audit logs are written.
        AuditDir string
        // Dialer performs SMTP sessions; nil → DeliverOnce returns ErrNoDialer.
        Dialer smtpdial.Dialer
        // Resolver looks up MX hosts; nil uses a static/direct fallback.
        Resolver mxresolve.Resolver
        // DirectHost forces delivery to a fixed host (lab / smarthost).
        DirectHost string
        // DirectPort SMTP port for DirectHost (default 25).
        DirectPort int
        // DKIM enables signing when Domain/Selector/PrivateKeyPEM set.
        DKIM DomainKey
        // DefaultBackoff is used when classifying temporary failures.
        DefaultBackoff time.Duration
        // MaxAttempts before permanent bounce on temp failures.
        MaxAttempts int
        // Clock overrides time for tests.
        Now func() time.Time
        // Auditor overrides file auditor (tests).
        Auditor audit.Logger
}

// DomainKey holds DKIM signing material.
type DomainKey struct {
        Domain         string
        Selector       string
        PrivateKeyPEM  string
        Headers        []string
        Canonicalization string
}

func (o Options) withDefaults() Options {
        if o.MaxQueue <= 0 {
                o.MaxQueue = 1024
        }
        if o.DirectPort <= 0 {
                o.DirectPort = 25
        }
        if o.DefaultBackoff <= 0 {
                o.DefaultBackoff = 30 * time.Second
        }
        if o.MaxAttempts <= 0 {
                o.MaxAttempts = 8
        }
        if o.Now == nil {
                o.Now = time.Now
        }
        if o.Resolver == nil {
                o.Resolver = mxresolve.Static{}
        }
        if len(o.DKIM.Headers) == 0 {
                o.DKIM.Headers = []string{"from", "to", "subject", "date", "mime-version", "content-type"}
        }
        if o.DKIM.Canonicalization == "" {
                o.DKIM.Canonicalization = "relaxed/relaxed"
        }
        return o
}
