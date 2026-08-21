package backoff

import "time"

// Exponential returns min(cap, base * 2^(attempt-1)) for attempt >= 1.
func Exponential(attempt int, base, cap time.Duration) time.Duration {
        if attempt < 1 {
                attempt = 1
        }
        if base <= 0 {
                base = time.Second
        }
        d := base
        for i := 1; i < attempt; i++ {
                if d >= cap || d > cap/2 {
                        return cap
                }
                d *= 2
        }
        if d > cap {
                return cap
        }
        return d
}

// Decorrelated jitter helper (deterministic mix without rand for tests).
func Decorrelated(prev, base, cap time.Duration, salt int64) time.Duration {
        if prev <= 0 {
                prev = base
        }
        next := base + time.Duration((salt%int64(prev)+int64(prev))%int64(prev+1))
        if next < base {
                next = base
        }
        if next > cap {
                return cap
        }
        return next
}

// Schedule builds a list of delays for n attempts.
func Schedule(n int, base, cap time.Duration) []time.Duration {
        if n < 0 {
                n = 0
        }
        out := make([]time.Duration, n)
        for i := 0; i < n; i++ {
                out[i] = Exponential(i+1, base, cap)
        }
        return out
}
