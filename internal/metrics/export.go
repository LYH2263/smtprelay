package metrics

import (
	"fmt"
	"strings"
)

// PrometheusText renders a minimal exposition format.
func (r *Registry) PrometheusText(prefix string) string {
	if prefix == "" {
		prefix = "smtprelay"
	}
	s := r.Snapshot()
	var b strings.Builder
	write := func(name string, v uint64) {
		fmt.Fprintf(&b, "%s_%s %d\n", prefix, name, v)
	}
	write("submitted_total", s.Submitted)
	write("delivered_total", s.Delivered)
	write("acked_total", s.Acked)
	write("bounced_total", s.Bounced)
	write("deferred_total", s.Deferred)
	write("peeked_total", s.Peeked)
	write("signed_total", s.Signed)
	write("persisted_total", s.Persisted)
	return b.String()
}
