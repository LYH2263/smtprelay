package backoff

import (
        "testing"
        "time"
)

func TestExponentialCap(t *testing.T) {
        d := Exponential(10, time.Second, 10*time.Second)
        if d != 10*time.Second {
                t.Fatalf("got %v", d)
        }
}
