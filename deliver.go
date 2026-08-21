package smtprelay

import (
	"context"
	"fmt"
	"strings"

	"github.com/LYH2263/go-smtprelay/internal/smtpdial"
)

// DeliverOnce peeks one message, signs, dials, and Ack/Nack accordingly.
func (r *Relay) DeliverOnce(ctx context.Context) (DeliveryResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	env, err := r.Peek()
	if err != nil {
		return DeliveryResult{}, err
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return DeliveryResult{}, ErrClosed
	}
	r.markInFlightLocked(env.ID)
	dialer := r.opts.Dialer
	r.mu.Unlock()

	if dialer == nil {
		_ = r.restorePending(env.ID)
		return DeliveryResult{MessageID: env.ID, AttemptedAt: r.now()}, ErrNoDialer
	}

	if err := EnsureMIME(env); err != nil {
		_ = r.restorePending(env.ID)
		return DeliveryResult{MessageID: env.ID}, err
	}
	if err := r.SignDKIM(env); err != nil {
		_ = r.restorePending(env.ID)
		return DeliveryResult{MessageID: env.ID}, err
	}

	host, port, err := r.resolveTarget(env)
	if err != nil {
		res := DeliveryResult{
			MessageID: env.ID, Code: 0, Permanent: false, Temporary: true,
			AttemptedAt: r.now(), Error: err.Error(),
		}
		_ = r.Nack(env.ID, res)
		return res, err
	}

	res, err := r.DialAndSend(ctx, dialer, host, port, env)
	res.MessageID = env.ID
	res.AttemptedAt = r.now()
	res.TargetHost = host
	_ = r.auditor.LogDelivery(env.ID, host, res.Code, res.Response, err)

	if err == nil && res.Code >= 200 && res.Code < 300 {
		if ackErr := r.Ack(env.ID); ackErr != nil {
			return res, ackErr
		}
		r.metrics.IncDelivered()
		return res, nil
	}

	if err != nil {
		if res.Error == "" {
			res.Error = err.Error()
		}

		res.Temporary = true
	}
	if nackErr := r.Nack(env.ID, res); nackErr != nil {
		return res, nackErr
	}

	if err != nil {
		return res, err
	}
	return res, fmt.Errorf("%s", res.Response)
}

// DialAndSend performs one SMTP transaction against host:port.
func (r *Relay) DialAndSend(ctx context.Context, dialer smtpdial.Dialer, host string, port int, env *Envelope) (DeliveryResult, error) {
	if dialer == nil {
		return DeliveryResult{}, ErrNoDialer
	}
	if err := ctx.Err(); err != nil {
		return DeliveryResult{}, err
	}
	// Handshake pacing must honor ctx (lab dialers may Sleep).
	select {
	case <-ctx.Done():
		return DeliveryResult{}, ctx.Err()
	default:
	}

	to := env.RecipientAddresses()
	code, resp, err := dialer.Send(ctx, smtpdial.Request{
		Host:     host,
		Port:     port,
		MailFrom: env.MailFrom,
		RcptTo:   to,
		Data:     env.RawBody,
	})
	res := DeliveryResult{Code: code, Response: resp, TargetHost: host}
	if err != nil {

		if code >= 500 && code < 600 {
			return res, fmt.Errorf("smtp %d %s: %v", code, resp, err)
		}
		if code >= 400 && code < 500 {
			return res, fmt.Errorf("smtp %d %s: %v", code, resp, err)
		}
		return res, err
	}
	if code >= 500 {
		return res, fmt.Errorf("smtp %d %s", code, resp)
	}
	if code >= 400 {
		return res, fmt.Errorf("smtp %d %s", code, resp)
	}
	return res, nil
}

func (r *Relay) resolveTarget(env *Envelope) (string, int, error) {
	if r.opts.DirectHost != "" {
		return r.opts.DirectHost, r.opts.DirectPort, nil
	}
	if len(env.Recipients) == 0 {
		return "", 0, ErrInvalidEnv
	}
	addr := env.Recipients[0].Address
	at := strings.LastIndex(addr, "@")
	if at < 0 || at == len(addr)-1 {
		return "", 0, ErrInvalidEnv
	}
	domain := addr[at+1:]
	hosts, err := r.opts.Resolver.LookupMX(domain)
	if err != nil {
		return "", 0, err
	}
	if len(hosts) == 0 {
		return domain, 25, nil
	}
	return hosts[0], 25, nil
}

func (r *Relay) restorePending(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if env, ok := r.byID[id]; ok {
		if env.State == StateInFlight {
			env.State = StatePending
		}
	}
	return nil
}
