package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSON(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		data           interface{}
		expectedStatus int
		validateJSON   bool
	}{
		{
			name:           "writes JSON with 200 status",
			status:         http.StatusOK,
			data:           map[string]string{"message": "success"},
			expectedStatus: http.StatusOK,
			validateJSON:   true,
		},
		{
			name:           "writes JSON with 201 status",
			status:         http.StatusCreated,
			data:           map[string]int{"id": 123},
			expectedStatus: http.StatusCreated,
			validateJSON:   true,
		},
		{
			name:   "handles complex nested data",
			status: http.StatusOK,
			data: map[string]interface{}{
				"tracks": []map[string]interface{}{
					{"id": 1, "name": "track1"},
					{"id": 2, "name": "track2"},
				},
				"total": 2,
			},
			expectedStatus: http.StatusOK,
			validateJSON:   true,
		},
		{
			name:           "handles nil data",
			status:         http.StatusNoContent,
			data:           nil,
			expectedStatus: http.StatusNoContent,
			validateJSON:   false,
		},
		{
			name:           "handles empty map",
			status:         http.StatusOK,
			data:           map[string]string{},
			expectedStatus: http.StatusOK,
			validateJSON:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			JSON(w, tt.status, tt.data)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check Content-Type header
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got %q", contentType)
			}

			// Validate JSON if expected
			if tt.validateJSON && tt.data != nil {
				var decoded interface{}
				if err := json.NewDecoder(w.Body).Decode(&decoded); err != nil {
					t.Fatalf("failed to decode JSON response: %v", err)
				}
			}
		})
	}
}

func TestJSONError(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		message        string
		expectedStatus int
		checkMessage   bool
	}{
		{
			name:           "writes 400 error",
			status:         http.StatusBadRequest,
			message:        "Invalid request",
			expectedStatus: http.StatusBadRequest,
			checkMessage:   true,
		},
		{
			name:           "writes 404 error",
			status:         http.StatusNotFound,
			message:        "Track not found",
			expectedStatus: http.StatusNotFound,
			checkMessage:   true,
		},
		{
			name:           "writes 500 error",
			status:         http.StatusInternalServerError,
			message:        "Internal server error",
			expectedStatus: http.StatusInternalServerError,
			checkMessage:   true,
		},
		{
			name:           "handles empty message",
			status:         http.StatusBadRequest,
			message:        "",
			expectedStatus: http.StatusBadRequest,
			checkMessage:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			JSONError(w, tt.status, tt.message)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check Content-Type header
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got %q", contentType)
			}

			// Decode and check error message
			if tt.checkMessage {
				var response map[string]string
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode JSON response: %v", err)
				}

				if errorMsg, ok := response["error"]; !ok {
					t.Error("response missing 'error' field")
				} else if errorMsg != tt.message {
					t.Errorf("expected error message %q, got %q", tt.message, errorMsg)
				}
			}
		})
	}
}

func TestJSONErrorFormat(t *testing.T) {
	w := httptest.NewRecorder()
	expectedMessage := "Test error message"

	JSONError(w, http.StatusInternalServerError, expectedMessage)

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	// Verify structure
	if len(response) != 1 {
		t.Errorf("expected response to have 1 field, got %d", len(response))
	}

	if _, ok := response["error"]; !ok {
		t.Error("response should only contain 'error' field")
	}
}
