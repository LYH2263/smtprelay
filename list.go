package smtprelay

// ListPending returns deep copies of pending/deferred/inflight envelopes.
func (r *Relay) ListPending() []*Envelope {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*Envelope, 0, len(r.order))
	for _, id := range r.order {
		env := r.byID[id]
		if env == nil {
			continue
		}
		if env.State != StatePending && env.State != StateDeferred && env.State != StateInFlight {
			continue
		}

		out = append(out, env.Clone())
	}
	return out
}

// ListBounces returns recent bounce records (copied).
func (r *Relay) ListBounces(limit int) []BounceRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 || limit > len(r.bounces) {
		limit = len(r.bounces)
	}
	src := r.bounces[len(r.bounces)-limit:]
	out := make([]BounceRecord, len(src))
	for i, b := range src {
		out[i] = b
		out[i].Recipients = append([]string(nil), b.Recipients...)
	}
	return out
}

// Get returns a deep copy of one message.
func (r *Relay) Get(id string) (*Envelope, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	env, ok := r.byID[id]
	if !ok {
		return nil, ErrNotFound
	}
	return env.Clone(), nil
}

// Stats returns queue counters.
func (r *Relay) Stats() QueueStats {
	r.mu.Lock()
	defer r.mu.Unlock()
	var s QueueStats
	for _, env := range r.byID {
		s.Total++
		switch env.State {
		case StatePending:
			s.Pending++
		case StateInFlight:
			s.InFlight++
		case StateDeferred:
			s.Deferred++
		case StateBounced:
			s.Bounced++
		case StateSent, StateAcked:
			s.Sent++
		}
	}
	return s
}
