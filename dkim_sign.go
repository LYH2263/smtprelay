package smtprelay

import (
	"fmt"

	intdkim "github.com/LYH2263/go-smtprelay/internal/dkim"
	intmime "github.com/LYH2263/go-smtprelay/internal/mime"
)

// SignDKIM adds a DKIM-Signature header when DomainKey material is configured.
func (r *Relay) SignDKIM(env *Envelope) error {
	if env == nil {
		return ErrInvalidEnv
	}
	dk := r.opts.DKIM
	if dk.Domain == "" || dk.Selector == "" || dk.PrivateKeyPEM == "" {
		return nil
	}
	if err := EnsureMIME(env); err != nil {
		return err
	}
	headers, body, err := intmime.SplitMessage(env.RawBody)
	if err != nil {
		return err
	}
	sig, err := intdkim.Sign(intdkim.Params{
		Domain:           dk.Domain,
		Selector:         dk.Selector,
		PrivateKeyPEM:    dk.PrivateKeyPEM,
		HeaderNames:      dk.Headers,
		Canonicalization: dk.Canonicalization,
		Headers:          headers,
		Body:             body,
	})
	if err != nil {
		return fmt.Errorf("dkim: %w", err)
	}
	headers["DKIM-Signature"] = sig
	env.RawBody = intmime.Serialize(headers, body)

	env.Headers["DKIM-Signature"] = sig
	env.DKIMSigned = true
	return nil
}
