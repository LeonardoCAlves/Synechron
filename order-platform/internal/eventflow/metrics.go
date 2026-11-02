package eventflow

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type Metrics struct {
	published           atomic.Uint64
	retried             atomic.Uint64
	deadLettered        atomic.Uint64
	consumed            atomic.Uint64
	duplicates          atomic.Uint64
	poisoned            atomic.Uint64
	outboxPending       atomic.Int64
	outboxOldestSeconds atomic.Int64
	outboxDeadLettered  atomic.Int64
}

func (metrics *Metrics) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	if _, err := fmt.Fprintf(w,
		"# TYPE order_platform_events_published_total counter\norder_platform_events_published_total %d\n"+
			"# TYPE order_platform_outbox_retries_total counter\norder_platform_outbox_retries_total %d\n"+
			"# TYPE order_platform_outbox_dead_lettered_total counter\norder_platform_outbox_dead_lettered_total %d\n"+
			"# TYPE order_platform_events_consumed_total counter\norder_platform_events_consumed_total %d\n"+
			"# TYPE order_platform_events_duplicate_total counter\norder_platform_events_duplicate_total %d\n"+
			"# TYPE order_platform_events_poisoned_total counter\norder_platform_events_poisoned_total %d\n"+
			"# TYPE order_platform_outbox_pending gauge\norder_platform_outbox_pending %d\n"+
			"# TYPE order_platform_outbox_oldest_pending_seconds gauge\norder_platform_outbox_oldest_pending_seconds %d\n"+
			"# TYPE order_platform_outbox_dead_lettered gauge\norder_platform_outbox_dead_lettered %d\n",
		metrics.published.Load(), metrics.retried.Load(), metrics.deadLettered.Load(),
		metrics.consumed.Load(), metrics.duplicates.Load(), metrics.poisoned.Load(),
		metrics.outboxPending.Load(), metrics.outboxOldestSeconds.Load(), metrics.outboxDeadLettered.Load()); err != nil {
		http.Error(w, "could not encode metrics", http.StatusInternalServerError)
	}
}

func (metrics *Metrics) setOutboxStats(pending int64, oldestSeconds int64, deadLettered int64) {
	metrics.outboxPending.Store(pending)
	metrics.outboxOldestSeconds.Store(oldestSeconds)
	metrics.outboxDeadLettered.Store(deadLettered)
}
