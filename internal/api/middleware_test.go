package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/costuff/doyou-music-analytics/pkg/logger"
)

func TestCORSMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		checkHeaders   map[string]string
	}{
		{
			name:           "sets CORS headers on GET request",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			checkHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
				"Access-Control-Allow-Headers": "Content-Type, Authorization",
			},
		},
		{
			name:           "handles OPTIONS preflight request",
			method:         http.MethodOptions,
			expectedStatus: http.StatusOK,
			checkHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
				"Access-Control-Allow-Headers": "Content-Type, Authorization",
			},
		},
		{
			name:           "sets CORS headers on POST request",
			method:         http.MethodPost,
			expectedStatus: http.StatusOK,
			checkHeaders: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler that returns 200 OK
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// Wrap with CORS middleware
			wrappedHandler := CORSMiddleware(handler)

			req := httptest.NewRequest(tt.method, "/test", nil)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check CORS headers
			for headerName, expectedValue := range tt.checkHeaders {
				actualValue := w.Header().Get(headerName)
				if actualValue != expectedValue {
					t.Errorf("header %s: expected %q, got %q", headerName, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestLoggingMiddleware(t *testing.T) {
	// Initialize logger for tests
	log := logger.New("info")

	tests := []struct {
		name             string
		method           string
		path             string
		handlerStatus    int
		handlerBody      string
		expectLogging    bool
	}{
		{
			name:          "logs GET request with 200 status",
			method:        http.MethodGet,
			path:          "/api/v1/tracks",
			handlerStatus: http.StatusOK,
			handlerBody:   `{"tracks":[]}`,
			expectLogging: true,
		},
		{
			name:          "logs POST request with 201 status",
			method:        http.MethodPost,
			path:          "/api/v1/tracks",
			handlerStatus: http.StatusCreated,
			handlerBody:   `{"id":1}`,
			expectLogging: true,
		},
		{
			name:          "logs request with 404 status",
			method:        http.MethodGet,
			path:          "/api/v1/tracks/999",
			handlerStatus: http.StatusNotFound,
			handlerBody:   `{"error":"not found"}`,
			expectLogging: true,
		},
		{
			name:          "logs request with 500 status",
			method:        http.MethodGet,
			path:          "/api/v1/stats",
			handlerStatus: http.StatusInternalServerError,
			handlerBody:   `{"error":"internal error"}`,
			expectLogging: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler that returns specified status
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.handlerStatus)
				w.Write([]byte(tt.handlerBody))
			})

			// Wrap with logging middleware
			wrappedHandler := LoggingMiddleware(log)(handler)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			// Verify the response
			if w.Code != tt.handlerStatus {
				t.Errorf("expected status %d, got %d", tt.handlerStatus, w.Code)
			}

			body := w.Body.String()
			if !strings.Contains(body, tt.handlerBody) {
				t.Errorf("expected body to contain %q, got %q", tt.handlerBody, body)
			}
		})
	}
}

func TestLoggingMiddleware_CapturesStatusCode(t *testing.T) {
	log := logger.New("info")

	tests := []struct {
		name           string
		handlerStatus  int
		writeBeforeStatus bool
	}{
		{
			name:           "captures status when WriteHeader called explicitly",
			handlerStatus:  http.StatusCreated,
			writeBeforeStatus: false,
		},
		{
			name:           "captures status when Write called without WriteHeader",
			handlerStatus:  http.StatusOK,
			writeBeforeStatus: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.writeBeforeStatus {
					// Write body first (implicit 200)
					w.Write([]byte("response"))
				} else {
					// Explicit status code
					w.WriteHeader(tt.handlerStatus)
					w.Write([]byte("response"))
				}
			})

			wrappedHandler := LoggingMiddleware(log)(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			expectedStatus := tt.handlerStatus
			if tt.writeBeforeStatus {
				expectedStatus = http.StatusOK
			}

			if w.Code != expectedStatus {
				t.Errorf("expected status %d, got %d", expectedStatus, w.Code)
			}
		})
	}
}

func TestMiddlewareChaining(t *testing.T) {
	log := logger.New("info")

	// Test that middleware can be chained together
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Apply both middleware
	wrappedHandler := CORSMiddleware(LoggingMiddleware(log)(handler))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// Verify CORS headers are present
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS headers not applied")
	}

	// Verify response is correct
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify body
	if !strings.Contains(w.Body.String(), "ok") {
		t.Error("response body incorrect")
	}
}
