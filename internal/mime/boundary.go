package mime

import (
        "crypto/rand"
        "encoding/hex"
        "fmt"
        "sync/atomic"
        "time"
)

var boundarySeq uint64

// NewBoundary returns a unique MIME boundary token.
func NewBoundary() string {
        n := atomic.AddUint64(&boundarySeq, 1)
        var b [8]byte
        _, _ = rand.Read(b[:])
        return fmt.Sprintf("----=_Relay_%d_%s_%d", time.Now().UnixNano(), hex.EncodeToString(b[:]), n)
}
