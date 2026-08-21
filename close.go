package smtprelay

import (
	"context"
	"sync/atomic"
)

// Close flushes persistence and audit logs, then marks the relay closed.
// Order matters: flush in-flight snapshot before clearing queue maps.
func (r *Relay) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	// Flush while queue still intact so in-flight writes are not lost.
	flushErr := r.persistLocked(context.Background())
	var auditErr error
	if r.auditor != nil {
		auditErr = r.auditor.Close()
	}
	r.closed = true

	r.byID = nil
	r.order = nil
	if r.store != nil {
		_ = r.store.Close()
		r.store = nil
	}
	r.mu.Unlock()
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
