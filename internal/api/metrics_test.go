package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsMiddlewareRecordsRequests(t *testing.T) {
	metrics := NewMetrics()
	handler := MetricsMiddleware(metrics)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "missing", http.StatusNotFound)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/missing", nil))

	w := httptest.NewRecorder()
	metrics.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := w.Body.String()
	if !strings.Contains(body, `http_requests_total{method="GET",status="404"} 1`) {
		t.Fatalf("expected request counter in metrics output, got:\n%s", body)
	}
	if !strings.Contains(body, "http_request_duration_seconds_sum") || !strings.Contains(body, "http_requests_in_flight 0") {
		t.Fatalf("expected duration and in-flight metrics, got:\n%s", body)
	}
}