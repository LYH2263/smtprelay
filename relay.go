package smtprelay

import (
        "sync"
        "time"

        "github.com/LYH2263/go-smtprelay/internal/audit"
        "github.com/LYH2263/go-smtprelay/internal/metrics"
        "github.com/LYH2263/go-smtprelay/internal/persist"
)

// Relay is the outbound SMTP relay facade.
type Relay struct {
        opts    Options
        mu      sync.Mutex
        closed  bool
        seq     uint64
        byID    map[string]*Envelope
        order   []string
        bounces []BounceRecord
        store   persister
        auditor audit.Logger
        metrics *metrics.Registry
}

// New constructs a Relay and optionally loads a queue snapshot.
func New(opts Options) (*Relay, error) {
        opts = opts.withDefaults()
        r := &Relay{
                opts:    opts,
                byID:    make(map[string]*Envelope),
                order:   make([]string, 0, 64),
                bounces: make([]BounceRecord, 0, 32),
                metrics: metrics.New(),
        }
        if opts.PersistPath != "" {
                st, err := persist.Open(opts.PersistPath)
                if err != nil {
                        return nil, err
                }
                r.store = st
                snap, err := st.Load()
                if err != nil {
                        return nil, err
                }
                for _, raw := range snap.Messages {
                        env := persistEnvelope(raw)
                        r.byID[env.ID] = env
                        r.order = append(r.order, env.ID)
                        if env.ID != "" {
                                r.bumpSeq(env.ID)
                        }
                }
        }
        if opts.Auditor != nil {
                r.auditor = opts.Auditor
        } else if opts.AuditDir != "" {
                a, err := audit.OpenDir(opts.AuditDir)
                if err != nil {
                        return nil, err
                }
                r.auditor = a
        } else {
                r.auditor = audit.Nop{}
        }
        return r, nil
}

func (r *Relay) bumpSeq(id string) {
        // best-effort: keep seq ahead of restored ids like msg-00042
        var n uint64
        if len(id) > 4 && id[:4] == "msg-" {
                for i := 4; i < len(id); i++ {
                        c := id[i]
                        if c < '0' || c > '9' {
                                return
                        }
                        n = n*10 + uint64(c-'0')
                }
                if n > r.seq {
                        r.seq = n
                }
        }
}

func (r *Relay) now() time.Time { return r.opts.Now() }

// Options returns a copy of configured options (dialer not cloned).
func (r *Relay) Options() Options { return r.opts }
