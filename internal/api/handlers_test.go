package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/costuff/doyou-music-analytics/pkg/logger"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	h := &Handlers{
		db:      nil, // Health doesn't use DB
		log:     logger.New("info"),
		version: "1.0.0-test",
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.HealthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check required fields
	requiredFields := []string{"status", "version", "timestamp", "service"}
	for _, field := range requiredFields {
		if _, ok := response[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}

	if response["version"] != "1.0.0-test" {
		t.Errorf("expected version 1.0.0-test, got %v", response["version"])
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

// TestGetTrackHandler_InvalidID tests invalid track ID handling
func TestGetTrackHandler_InvalidID(t *testing.T) {
	h := &Handlers{
		db:      nil,
		log:     logger.New("info"),
		version: "1.0.0",
	}

	tests := []struct {
		name string
		path string
	}{
		{"non-numeric ID", "/api/v1/tracks/abc"},
		{"empty ID", "/api/v1/tracks/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			h.GetTrackHandler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", w.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if _, ok := response["error"]; !ok {
				t.Error("expected error field in response")
			}
		})
	}
}

// TestSearchTracksHandler_MissingQuery tests missing query parameter
func TestSearchTracksHandler_MissingQuery(t *testing.T) {
	h := &Handlers{
		db:      nil,
		log:     logger.New("info"),
		version: "1.0.0",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/search", nil)
	w := httptest.NewRecorder()

	h.SearchTracksHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("expected error field in response")
	}
}
