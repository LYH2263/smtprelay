package smtprelay

import "errors"

var (
        ErrClosed       = errors.New("smtprelay: relay closed")
        ErrNoDialer     = errors.New("smtprelay: no dialer configured")
        ErrInvalidEnv   = errors.New("smtprelay: invalid envelope")
        ErrNotFound     = errors.New("smtprelay: message not found")
        ErrPermanent    = errors.New("smtprelay: permanent delivery failure")
        ErrTemporary    = errors.New("smtprelay: temporary delivery failure")
        ErrQueueFull    = errors.New("smtprelay: queue full")
        ErrPersist      = errors.New("smtprelay: persist failed")
        ErrAlreadyAcked = errors.New("smtprelay: already acked")
        ErrEmptyBody    = errors.New("smtprelay: empty body")
)
