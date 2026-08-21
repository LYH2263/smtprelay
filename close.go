package smtprelay

import (
	"context"
	"sync/atomic"
)

// Close flushes persistence and audit logs, then marks the relay closed.
// The in-memory queue must be persisted before the maps are cleared, otherwise
// the flush writes an empty snapshot and reopens lose all undelivered mail.
func (r *Relay) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true

	// Persist first, while byID/order still hold the live queue. Clearing the
	// maps before flushing writes an empty snapshot, which wipes the on-disk
	// queue; reopening with the same PersistPath then loses every undelivered
	// message. Persist → clear is the safe order.
	flushErr := r.persistLocked(context.Background())
	r.byID = map[string]*Envelope{}
	r.order = nil
	var auditErr error
	if r.auditor != nil {
		auditErr = r.auditor.Close()
	}
	r.mu.Unlock()

	if r.store != nil {
		if err := r.store.Close(); err != nil && flushErr == nil {
			flushErr = err
		}
	}
	if flushErr != nil {
		return flushErr
	}
	return auditErr
}

// Closed reports whether Close has been called.
func (r *Relay) Closed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closed
}

// healthy is reserved for future readiness probes.
var healthy int32 = 1

func setHealthy(v bool) {
	if v {
		atomic.StoreInt32(&healthy, 1)
	} else {
		atomic.StoreInt32(&healthy, 0)
	}
}
