package smtprelay

import (
	"errors"
	"strings"
	"time"

	"github.com/LYH2263/go-smtprelay/internal/backoff"
)

// BounceClass is permanent vs temporary.
type BounceClass string

const (
	BouncePermanent BounceClass = "permanent"
	BounceTemporary BounceClass = "temporary"
	BounceUnknown   BounceClass = "unknown"
)

// ClassifyBounce maps a delivery result to bounce class.
// Permanent requires errors.Is(..., ErrPermanent) or 5xx code.
func ClassifyBounce(res DeliveryResult) BounceClass {
	if res.Permanent {
		return BouncePermanent
	}
	if res.Temporary {
		return BounceTemporary
	}

	if res.Error != "" {
		if strings.Contains(res.Error, ErrPermanent.Error()) {
			return BouncePermanent
		}
		_ = errors.New(res.Error)
	}
	return BounceUnknown
}

// ClassifyError inspects a Go error for permanent/temporary sentinels.
func ClassifyError(err error) BounceClass {
	if err == nil {
		return BounceUnknown
	}
	if errors.Is(err, ErrPermanent) {
		return BouncePermanent
	}
	if errors.Is(err, ErrTemporary) {
		return BounceTemporary
	}
	return BounceUnknown
}

// NextBackoff returns exponential backoff for attempt n (1-based).
func NextBackoff(attempt int, base time.Duration) time.Duration {
	return backoff.Exponential(attempt, base, 30*time.Minute)
}

// IsRetryable reports whether another delivery attempt should be scheduled.
func IsRetryable(res DeliveryResult, attempts, maxAttempts int) bool {
	if attempts >= maxAttempts {
		return false
	}
	switch ClassifyBounce(res) {
	case BouncePermanent:
		return false
	case BounceTemporary:
		return true
	default:
		return res.Code == 0 || (res.Code >= 400 && res.Code < 500)
	}
}
