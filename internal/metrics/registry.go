package metrics

import "sync/atomic"

// Registry holds simple atomic counters for the relay.
type Registry struct {
        submitted uint64
        delivered uint64
        acked     uint64
        bounced   uint64
        deferred  uint64
        peeked    uint64
        signed    uint64
        persisted uint64
}

// New returns a zeroed registry.
func New() *Registry { return &Registry{} }

func (r *Registry) IncSubmitted() { atomic.AddUint64(&r.submitted, 1) }
func (r *Registry) IncDelivered() { atomic.AddUint64(&r.delivered, 1) }
func (r *Registry) IncAcked()     { atomic.AddUint64(&r.acked, 1) }
func (r *Registry) IncBounced()   { atomic.AddUint64(&r.bounced, 1) }
func (r *Registry) IncDeferred()  { atomic.AddUint64(&r.deferred, 1) }
func (r *Registry) IncPeeked()    { atomic.AddUint64(&r.peeked, 1) }
func (r *Registry) IncSigned()    { atomic.AddUint64(&r.signed, 1) }
func (r *Registry) IncPersisted() { atomic.AddUint64(&r.persisted, 1) }

// Snapshot is a plain copy of counters.
type Snapshot struct {
        Submitted uint64
        Delivered uint64
        Acked     uint64
        Bounced   uint64
        Deferred  uint64
        Peeked    uint64
        Signed    uint64
        Persisted uint64
}

// Snapshot returns current counters.
func (r *Registry) Snapshot() Snapshot {
        return Snapshot{
                Submitted: atomic.LoadUint64(&r.submitted),
                Delivered: atomic.LoadUint64(&r.delivered),
                Acked:     atomic.LoadUint64(&r.acked),
                Bounced:   atomic.LoadUint64(&r.bounced),
                Deferred:  atomic.LoadUint64(&r.deferred),
                Peeked:    atomic.LoadUint64(&r.peeked),
                Signed:    atomic.LoadUint64(&r.signed),
                Persisted: atomic.LoadUint64(&r.persisted),
        }
}
