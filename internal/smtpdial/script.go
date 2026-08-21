package smtpdial

import (
        "context"
        "fmt"
        "sync"
)

// ScriptedDialer returns sequential responses (tests / bounce scenarios).
type ScriptedDialer struct {
        mu    sync.Mutex
        steps []step
        idx   int
        Last  Request
}

type step struct {
        Code int
        Resp string
        Err  error
}

// Add queues a response.
func (s *ScriptedDialer) Add(code int, resp string, err error) {
        s.mu.Lock()
        defer s.mu.Unlock()
        s.steps = append(s.steps, step{Code: code, Resp: resp, Err: err})
}

func (s *ScriptedDialer) Send(ctx context.Context, req Request) (int, string, error) {
        if err := ctx.Err(); err != nil {
                return 0, "", err
        }
        s.mu.Lock()
        defer s.mu.Unlock()
        s.Last = req
        if s.idx >= len(s.steps) {
                return 250, "OK", nil
        }
        st := s.steps[s.idx]
        s.idx++
        return st.Code, st.Resp, st.Err
}

// FailPermanent is a convenience error for scripted 5xx.
func FailPermanent(code int, msg string) error {
        return fmt.Errorf("smtp permanent %d: %s", code, msg)
}
