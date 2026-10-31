package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/costuff/doyou-music-analytics/internal/api"
	"github.com/costuff/doyou-music-analytics/pkg/logger"
)

func TestRouteTrackByID_PreservesSubresourceWithQuery(t *testing.T) {
	handler := routeTrackByID(api.NewHandlers(nil, logger.New("error"), "test"))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/not-an-id/similar?n=15", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid track id") {
		t.Fatalf("query string selected wrong route: status=%d body=%s", w.Code, w.Body.String())
	}
}