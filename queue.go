package smtprelay

import (
	"context"
	"fmt"

	"github.com/LYH2263/go-smtprelay/internal/clone"
	"github.com/LYH2263/go-smtprelay/internal/persist"
)

// Submit enqueues an envelope (clones body/headers/recipients).
func (r *Relay) Submit(env *Envelope) (string, error) {
	return r.SubmitContext(context.Background(), env)
}

// SubmitContext enqueues with cancellation around validation/persist.
func (r *Relay) SubmitContext(ctx context.Context, env *Envelope) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if env == nil {
		return "", ErrInvalidEnv
	}
	if err := env.Validate(); err != nil {
		return "", err
	}

	stored := env.Clone()

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return "", ErrClosed
	}
	if len(r.byID) >= r.opts.MaxQueue {
		return "", ErrQueueFull
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	r.seq++
	id := stored.ID
	if id == "" {
		id = fmt.Sprintf("msg-%05d", r.seq)
		stored.ID = id
	}
	if stored.CreatedAt.IsZero() {
		stored.CreatedAt = r.now()
	}
	stored.State = StatePending
	stored.NextAttempt = r.now()
	if stored.Headers == nil {
		stored.Headers = map[string]string{}
	}
	if _, ok := stored.Headers["Message-Id"]; !ok {
		stored.Headers["Message-Id"] = fmt.Sprintf("<%s@local.relay>", id)
	}

	r.byID[id] = stored
	r.order = append(r.order, id)
	r.metrics.IncSubmitted()

	if err := r.persistLocked(ctx); err != nil {
		delete(r.byID, id)
		r.order = r.order[:len(r.order)-1]
		return "", err
	}
	return id, nil
}

// Peek returns the next deliverable pending/deferred message without removing it.
func (r *Relay) Peek() (*Envelope, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, ErrClosed
	}
	now := r.now()
	for _, id := range r.order {
		env := r.byID[id]
		if env == nil {
			continue
		}
		if env.State != StatePending && env.State != StateDeferred {
			continue
		}
		if env.NextAttempt.After(now) {
			continue
		}
		return env.Clone(), nil
	}
	return nil, ErrNotFound
}

// Ack marks a message as sent and removes it from the active queue.
func (r *Relay) Ack(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ErrClosed
	}
	env, ok := r.byID[id]
	if !ok {
		return ErrNotFound
	}
	if env.State == StateAcked {
		return ErrAlreadyAcked
	}
	env.State = StateSent
	if err := r.persistLocked(context.Background()); err != nil {
		env.State = StateInFlight
		return err
	}
	delete(r.byID, id)
	r.removeOrderLocked(id)
	r.metrics.IncAcked()
	return nil
}

// Nack records a delivery failure; permanent → bounce, temporary → defer.
func (r *Relay) Nack(id string, result DeliveryResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ErrClosed
	}
	env, ok := r.byID[id]
	if !ok {
		return ErrNotFound
	}
	env.Attempts++
	env.LastError = result.Error
	if env.LastError == "" {
		env.LastError = result.Response
	}
	class := ClassifyBounce(result)
	if class == BouncePermanent || env.Attempts >= r.opts.MaxAttempts {
		env.State = StateBounced
		rec := BounceRecord{
			MessageID:  id,
			At:         r.now(),
			Code:       result.Code,
			Class:      string(class),
			Detail:     env.LastError,
			Recipients: env.RecipientAddresses(),
		}
		r.bounces = append(r.bounces, rec)
		if len(r.bounces) > 200 {
			r.bounces = r.bounces[len(r.bounces)-200:]
		}
		r.metrics.IncBounced()
		if err := r.persistLocked(context.Background()); err != nil {
			return err
		}
		delete(r.byID, id)
		r.removeOrderLocked(id)
		return nil
	}
	env.State = StateDeferred
	delay := NextBackoff(env.Attempts, r.opts.DefaultBackoff)
	env.NextAttempt = r.now().Add(delay)
	r.metrics.IncDeferred()
	return r.persistLocked(context.Background())
}

func (r *Relay) removeOrderLocked(id string) {
	out := r.order[:0]
	for _, x := range r.order {
		if x != id {
			out = append(out, x)
		}
	}
	r.order = out
}

func (r *Relay) persistLocked(ctx context.Context) error {
	if r.store == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	snap := persist.Snapshot{SavedAt: r.now().UTC(), Messages: make([]persist.Message, 0, len(r.byID))}
	for _, id := range r.order {
		env := r.byID[id]
		if env == nil {
			continue
		}
		snap.Messages = append(snap.Messages, envelopePersist(env))
	}
	if err := r.store.Save(ctx, snap); err != nil {
		return fmt.Errorf("%w: %v", ErrPersist, err)
	}
	return nil
}

func envelopePersist(e *Envelope) persist.Message {
	recips := make([]persist.Recipient, len(e.Recipients))
	for i, r := range e.Recipients {
		recips[i] = persist.Recipient{Address: r.Address, Name: r.Name}
	}
	parts := make([]persist.MimePart, len(e.MimeParts))
	for i, p := range e.MimeParts {
		parts[i] = persist.MimePart{
			ContentType: p.ContentType, Charset: p.Charset, Disposition: p.Disposition,
			Filename: p.Filename, ContentID: p.ContentID, Data: clone.Bytes(p.Data),
		}
	}
	return persist.Message{
		ID: e.ID, MailFrom: e.MailFrom, Recipients: recips, Subject: e.Subject,
		Headers: clone.HeaderMap(e.Headers), RawBody: clone.Bytes(e.RawBody), MimeParts: parts,
		CreatedAt: e.CreatedAt, Attempts: e.Attempts, NextAttempt: e.NextAttempt,
		State: string(e.State), LastError: e.LastError, DKIMSigned: e.DKIMSigned,
	}
}

func persistEnvelope(m persist.Message) *Envelope {
	recips := make([]Recipient, len(m.Recipients))
	for i, r := range m.Recipients {
		recips[i] = Recipient{Address: r.Address, Name: r.Name}
	}
	parts := make([]MimePart, len(m.MimeParts))
	for i, p := range m.MimeParts {
		parts[i] = MimePart{
			ContentType: p.ContentType, Charset: p.Charset, Disposition: p.Disposition,
			Filename: p.Filename, ContentID: p.ContentID, Data: clone.Bytes(p.Data),
		}
	}
	return &Envelope{
		ID: m.ID, MailFrom: m.MailFrom, Recipients: recips, Subject: m.Subject,
		Headers: clone.HeaderMap(m.Headers), RawBody: clone.Bytes(m.RawBody), MimeParts: parts,
		CreatedAt: m.CreatedAt, Attempts: m.Attempts, NextAttempt: m.NextAttempt,
		State: MessageState(m.State), LastError: m.LastError, DKIMSigned: m.DKIMSigned,
	}
}

func (r *Relay) markInFlightLocked(id string) {
	if env, ok := r.byID[id]; ok {
		env.State = StateInFlight
	}
}
