package api

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics records bounded-cardinality HTTP request counters and duration totals.
type Metrics struct {
	mu             sync.Mutex
	requests       map[requestLabel]uint64
	requestCount   atomic.Uint64
	durationNanos  atomic.Uint64
	inFlight       atomic.Int64
}

type requestLabel struct {
	method string
	status int
}

func NewMetrics() *Metrics {
	return &Metrics{requests: make(map[requestLabel]uint64)}
}

func MetricsMiddleware(metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			metrics.inFlight.Add(1)
			defer metrics.inFlight.Add(-1)

			observed := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(observed, r)
			metrics.requestCount.Add(1)
			metrics.durationNanos.Add(uint64(time.Since(started)))
			method := r.Method
			switch method {
			case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions:
			default:
				method = "OTHER"
			}
			label := requestLabel{method: method, status: observed.statusCode}
			metrics.mu.Lock()
			metrics.requests[label]++
			metrics.mu.Unlock()
		})
	}
}

func (m *Metrics) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(w, "# HELP http_requests_total Total HTTP requests handled.\n# TYPE http_requests_total counter\n")
	m.mu.Lock()
	for label, count := range m.requests {
		fmt.Fprintf(w, "http_requests_total{method=%q,status=%q} %d\n", label.method, strconv.Itoa(label.status), count)
	}
	m.mu.Unlock()
	fmt.Fprintf(w, "# HELP http_request_duration_seconds_sum Cumulative HTTP request duration.\n# TYPE http_request_duration_seconds_sum counter\nhttp_request_duration_seconds_sum %g\n", float64(m.durationNanos.Load())/float64(time.Second))
	fmt.Fprintf(w, "# HELP http_requests_in_flight Current in-flight HTTP requests.\n# TYPE http_requests_in_flight gauge\nhttp_requests_in_flight %d\n", m.inFlight.Load())
}
