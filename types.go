package smtprelay

import "time"

// MessageState is the queue lifecycle of an outbound mail.
type MessageState string

const (
        StatePending  MessageState = "pending"
        StateInFlight MessageState = "inflight"
        StateDeferred MessageState = "deferred"
        StateBounced  MessageState = "bounced"
        StateSent     MessageState = "sent"
        StateAcked    MessageState = "acked"
)

// DeliveryResult summarizes one DialAndSend attempt.
type DeliveryResult struct {
        MessageID   string
        Code        int
        Response    string
        Permanent   bool
        Temporary   bool
        AttemptedAt time.Time
        TargetHost  string
        Error       string
}

// QueueStats is a snapshot for the admin UI / metrics.
type QueueStats struct {
        Pending  int
        InFlight int
        Deferred int
        Bounced  int
        Sent     int
        Total    int
}

// BounceRecord is retained for recent failure inspection.
type BounceRecord struct {
        MessageID string
        At        time.Time
        Code      int
        Class     string
        Detail    string
        Recipients []string
}
