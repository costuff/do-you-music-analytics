package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/costuff/doyou-music-analytics/internal/db"
	"github.com/costuff/doyou-music-analytics/pkg/logger"
)

// mockTrackStore implements TrackStore for testing.
type mockTrackStore struct {
	getTrackStatsFunc                func(ctx context.Context) (map[string]int64, error)
	listTracksFunc                   func(ctx context.Context, limit, offset int, status string) ([]db.TrackWithFeatures, int64, error)
	getTrackByIDFunc                 func(ctx context.Context, trackID int64) (*db.TrackWithFeatures, error)
	searchTracksFunc                 func(ctx context.Context, query string, limit int) ([]db.TrackWithFeatures, error)
	getFailedTracksFunc              func(ctx context.Context, limit int) ([]db.TrackWithFeatures, error)
	getRecentCompletedTracksFunc     func(ctx context.Context, limit int) ([]db.TrackWithFeatures, error)
	getEmotionsFunc                  func(ctx context.Context, trackID int64) (*db.TrackEmotions, error)
	getEmotionsDistributionFunc      func(ctx context.Context) ([]db.EmotionDistribution, error)
}

func (m *mockTrackStore) GetTrackStats(ctx context.Context) (map[string]int64, error) {
	return m.getTrackStatsFunc(ctx)
}
func (m *mockTrackStore) ListTracks(ctx context.Context, limit, offset int, status string) ([]db.TrackWithFeatures, int64, error) {
	return m.listTracksFunc(ctx, limit, offset, status)
}
func (m *mockTrackStore) GetTrackByID(ctx context.Context, trackID int64) (*db.TrackWithFeatures, error) {
	return m.getTrackByIDFunc(ctx, trackID)
}
func (m *mockTrackStore) SearchTracks(ctx context.Context, query string, limit int) ([]db.TrackWithFeatures, error) {
	return m.searchTracksFunc(ctx, query, limit)
}
func (m *mockTrackStore) GetFailedTracks(ctx context.Context, limit int) ([]db.TrackWithFeatures, error) {
	return m.getFailedTracksFunc(ctx, limit)
}
func (m *mockTrackStore) GetRecentCompletedTracks(ctx context.Context, limit int) ([]db.TrackWithFeatures, error) {
	return m.getRecentCompletedTracksFunc(ctx, limit)
}
func (m *mockTrackStore) GetEmotions(ctx context.Context, trackID int64) (*db.TrackEmotions, error) {
	if m.getEmotionsFunc != nil {
		return m.getEmotionsFunc(ctx, trackID)
	}
	return nil, nil
}
func (m *mockTrackStore) GetEmotionsDistribution(ctx context.Context) ([]db.EmotionDistribution, error) {
	if m.getEmotionsDistributionFunc != nil {
		return m.getEmotionsDistributionFunc(ctx)
	}
	return nil, nil
}

func newTestHandlers(store TrackStore) *Handlers {
	return &Handlers{db: store, log: logger.New("info"), version: "1.0.0-test"}
}

// ---- HealthHandler ----

func TestHealthHandler(t *testing.T) {
	h := newTestHandlers(nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.HealthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	for _, field := range []string{"status", "version", "timestamp", "service"} {
		if _, ok := resp[field]; !ok {
			t.Errorf("missing field: %s", field)
		}
	}
	if resp["version"] != "1.0.0-test" {
		t.Errorf("version = %v, want 1.0.0-test", resp["version"])
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", w.Header().Get("Content-Type"))
	}
}

// ---- StatsHandler ----

func TestStatsHandler_Success(t *testing.T) {
	store := &mockTrackStore{
		getTrackStatsFunc: func(ctx context.Context) (map[string]int64, error) {
			return map[string]int64{"total": 100, "pending": 50, "analyzing": 5, "completed": 40, "failed": 5}, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	w := httptest.NewRecorder()
	h.StatsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	for _, field := range []string{"total", "pending", "analyzing", "completed", "failed"} {
		if _, ok := resp[field]; !ok {
			t.Errorf("missing field: %s", field)
		}
	}
}

func TestStatsHandler_DBError(t *testing.T) {
	store := &mockTrackStore{
		getTrackStatsFunc: func(ctx context.Context) (map[string]int64, error) {
			return nil, errors.New("db connection lost")
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	w := httptest.NewRecorder()
	h.StatsHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if _, ok := resp["error"]; !ok {
		t.Error("expected error field in response")
	}
}

// ---- ListTracksHandler ----

func TestListTracksHandler_Success(t *testing.T) {
	tracks := []db.TrackWithFeatures{
		{ID: 1, FileName: "track1.mp3", AnalysisStatus: "completed"},
		{ID: 2, FileName: "track2.mp3", AnalysisStatus: "pending"},
	}
	store := &mockTrackStore{
		listTracksFunc: func(ctx context.Context, limit, offset int, status string) ([]db.TrackWithFeatures, int64, error) {
			return tracks, 2, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks", nil)
	w := httptest.NewRecorder()
	h.ListTracksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	for _, field := range []string{"tracks", "total", "limit", "offset"} {
		if _, ok := resp[field]; !ok {
			t.Errorf("missing field: %s", field)
		}
	}
}

func TestListTracksHandler_WithLimitOffset(t *testing.T) {
	var capturedLimit, capturedOffset int
	var capturedStatus string
	store := &mockTrackStore{
		listTracksFunc: func(ctx context.Context, limit, offset int, status string) ([]db.TrackWithFeatures, int64, error) {
			capturedLimit = limit
			capturedOffset = offset
			capturedStatus = status
			return []db.TrackWithFeatures{}, 0, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks?limit=10&offset=20&status=completed", nil)
	w := httptest.NewRecorder()
	h.ListTracksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if capturedLimit != 10 {
		t.Errorf("limit = %d, want 10", capturedLimit)
	}
	if capturedOffset != 20 {
		t.Errorf("offset = %d, want 20", capturedOffset)
	}
	if capturedStatus != "completed" {
		t.Errorf("status = %q, want completed", capturedStatus)
	}
}

func TestListTracksHandler_InvalidLimit_UsesDefault(t *testing.T) {
	var capturedLimit int
	store := &mockTrackStore{
		listTracksFunc: func(ctx context.Context, limit, offset int, status string) ([]db.TrackWithFeatures, int64, error) {
			capturedLimit = limit
			return []db.TrackWithFeatures{}, 0, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks?limit=999", nil)
	w := httptest.NewRecorder()
	h.ListTracksHandler(w, req)

	// limit > 200 should fall back to default (50)
	if capturedLimit != 50 {
		t.Errorf("limit = %d, want 50 (default enforced for >200)", capturedLimit)
	}
}

func TestListTracksHandler_DBError(t *testing.T) {
	store := &mockTrackStore{
		listTracksFunc: func(ctx context.Context, limit, offset int, status string) ([]db.TrackWithFeatures, int64, error) {
			return nil, 0, errors.New("query failed")
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks", nil)
	w := httptest.NewRecorder()
	h.ListTracksHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// ---- GetTrackHandler ----

func TestGetTrackHandler_InvalidID_NonNumeric(t *testing.T) {
	h := newTestHandlers(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/abc", nil)
	w := httptest.NewRecorder()
	h.GetTrackHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetTrackHandler_EmptyID(t *testing.T) {
	h := newTestHandlers(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/", nil)
	w := httptest.NewRecorder()
	h.GetTrackHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetTrackHandler_Success(t *testing.T) {
	now := time.Now()
	track := &db.TrackWithFeatures{
		ID: 42, FileName: "song.mp3", AnalysisStatus: "completed",
		DownloadedAt: &now,
	}
	store := &mockTrackStore{
		getTrackByIDFunc: func(ctx context.Context, trackID int64) (*db.TrackWithFeatures, error) {
			if trackID != 42 {
				return nil, errors.New("not found")
			}
			return track, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/42", nil)
	w := httptest.NewRecorder()
	h.GetTrackHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp db.TrackWithFeatures
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.ID != 42 {
		t.Errorf("ID = %d, want 42", resp.ID)
	}
}

func TestGetTrackHandler_NotFound(t *testing.T) {
	store := &mockTrackStore{
		getTrackByIDFunc: func(ctx context.Context, trackID int64) (*db.TrackWithFeatures, error) {
			return nil, errors.New("failed to get track: no rows in result set")
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/999", nil)
	w := httptest.NewRecorder()
	h.GetTrackHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ---- SearchTracksHandler ----

func TestSearchTracksHandler_MissingQuery(t *testing.T) {
	h := newTestHandlers(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/search", nil)
	w := httptest.NewRecorder()
	h.SearchTracksHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if _, ok := resp["error"]; !ok {
		t.Error("expected error field in response")
	}
}

func TestSearchTracksHandler_Success(t *testing.T) {
	results := []db.TrackWithFeatures{{ID: 1, FileName: "found.mp3", AnalysisStatus: "completed"}}
	store := &mockTrackStore{
		searchTracksFunc: func(ctx context.Context, query string, limit int) ([]db.TrackWithFeatures, error) {
			return results, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/search?q=found", nil)
	w := httptest.NewRecorder()
	h.SearchTracksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp["query"] != "found" {
		t.Errorf("query = %v, want 'found'", resp["query"])
	}
	if count, ok := resp["count"].(float64); !ok || count != 1 {
		t.Errorf("count = %v, want 1", resp["count"])
	}
}

func TestSearchTracksHandler_DBError(t *testing.T) {
	store := &mockTrackStore{
		searchTracksFunc: func(ctx context.Context, query string, limit int) ([]db.TrackWithFeatures, error) {
			return nil, errors.New("search failed")
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/search?q=test", nil)
	w := httptest.NewRecorder()
	h.SearchTracksHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestSearchTracksHandler_CustomLimit(t *testing.T) {
	var capturedLimit int
	store := &mockTrackStore{
		searchTracksFunc: func(ctx context.Context, query string, limit int) ([]db.TrackWithFeatures, error) {
			capturedLimit = limit
			return []db.TrackWithFeatures{}, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/search?q=test&limit=25", nil)
	w := httptest.NewRecorder()
	h.SearchTracksHandler(w, req)

	if capturedLimit != 25 {
		t.Errorf("limit = %d, want 25", capturedLimit)
	}
}

// ---- FailedTracksHandler ----

func TestFailedTracksHandler_Success(t *testing.T) {
	errMsg := "librosa decode error"
	tracks := []db.TrackWithFeatures{
		{ID: 1, FileName: "bad.mp3", AnalysisStatus: "failed", AnalysisError: &errMsg},
	}
	store := &mockTrackStore{
		getFailedTracksFunc: func(ctx context.Context, limit int) ([]db.TrackWithFeatures, error) {
			return tracks, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/failed", nil)
	w := httptest.NewRecorder()
	h.FailedTracksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if _, ok := resp["failed_tracks"]; !ok {
		t.Error("missing failed_tracks field")
	}
	if count, ok := resp["count"].(float64); !ok || count != 1 {
		t.Errorf("count = %v, want 1", resp["count"])
	}
}

func TestFailedTracksHandler_DBError(t *testing.T) {
	store := &mockTrackStore{
		getFailedTracksFunc: func(ctx context.Context, limit int) ([]db.TrackWithFeatures, error) {
			return nil, errors.New("query error")
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/failed", nil)
	w := httptest.NewRecorder()
	h.FailedTracksHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestFailedTracksHandler_CustomLimit(t *testing.T) {
	var capturedLimit int
	store := &mockTrackStore{
		getFailedTracksFunc: func(ctx context.Context, limit int) ([]db.TrackWithFeatures, error) {
			capturedLimit = limit
			return []db.TrackWithFeatures{}, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/failed?limit=10", nil)
	w := httptest.NewRecorder()
	h.FailedTracksHandler(w, req)

	if capturedLimit != 10 {
		t.Errorf("limit = %d, want 10", capturedLimit)
	}
}

// ---- DetailedStatsHandler ----

func TestDetailedStatsHandler_Success(t *testing.T) {
	now := time.Now()
	store := &mockTrackStore{
		getTrackStatsFunc: func(ctx context.Context) (map[string]int64, error) {
			return map[string]int64{"total": 200, "pending": 100, "analyzing": 2, "completed": 90, "failed": 8}, nil
		},
		getRecentCompletedTracksFunc: func(ctx context.Context, limit int) ([]db.TrackWithFeatures, error) {
			return []db.TrackWithFeatures{
				{ID: 1, AnalyzedAt: &now, DownloadedAt: &now},
				{ID: 2, AnalyzedAt: &now, DownloadedAt: &now},
			}, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/detailed", nil)
	w := httptest.NewRecorder()
	h.DetailedStatsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	for _, field := range []string{"total", "completed", "completion_percentage", "avg_processing_time_sec", "tracks_per_hour"} {
		if _, ok := resp[field]; !ok {
			t.Errorf("missing field: %s", field)
		}
	}
	// 90 completed out of 200 total = 45%
	if pct, ok := resp["completion_percentage"].(string); !ok || pct != "45.00" {
		t.Errorf("completion_percentage = %v, want 45.00", resp["completion_percentage"])
	}
}

func TestDetailedStatsHandler_ZeroTotal(t *testing.T) {
	store := &mockTrackStore{
		getTrackStatsFunc: func(ctx context.Context) (map[string]int64, error) {
			return map[string]int64{"total": 0, "pending": 0, "analyzing": 0, "completed": 0, "failed": 0}, nil
		},
		getRecentCompletedTracksFunc: func(ctx context.Context, limit int) ([]db.TrackWithFeatures, error) {
			return []db.TrackWithFeatures{}, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/detailed", nil)
	w := httptest.NewRecorder()
	h.DetailedStatsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if pct, ok := resp["completion_percentage"].(string); !ok || pct != "0.00" {
		t.Errorf("completion_percentage = %v, want 0.00", resp["completion_percentage"])
	}
}

func TestDetailedStatsHandler_DBError(t *testing.T) {
	store := &mockTrackStore{
		getTrackStatsFunc: func(ctx context.Context) (map[string]int64, error) {
			return nil, errors.New("stats query failed")
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/detailed", nil)
	w := httptest.NewRecorder()
	h.DetailedStatsHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// ── GetTrackEmotionsHandler ──────────────────────────────────────────────────

func TestGetTrackEmotionsHandler_Found(t *testing.T) {
	wantEmotions := &db.TrackEmotions{
		TrackID:    7,
		Tension:    0.45,
		Melancholy: 0.30,
		Euphoria:   0.60,
		Aggression: 0.25,
		Serenity:   0.55,
		Nostalgia:  0.40,
		Sensuality: 0.50,
		Power:      0.70,
	}
	store := &mockTrackStore{
		getEmotionsFunc: func(_ context.Context, trackID int64) (*db.TrackEmotions, error) {
			if trackID != 7 {
				return nil, nil
			}
			return wantEmotions, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/7/emotions", nil)
	w := httptest.NewRecorder()
	h.GetTrackEmotionsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var got db.TrackEmotions
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if got.TrackID != 7 {
		t.Errorf("TrackID = %d, want 7", got.TrackID)
	}
	if got.Euphoria != wantEmotions.Euphoria {
		t.Errorf("Euphoria = %f, want %f", got.Euphoria, wantEmotions.Euphoria)
	}
}

func TestGetTrackEmotionsHandler_NotFound(t *testing.T) {
	store := &mockTrackStore{
		getEmotionsFunc: func(_ context.Context, _ int64) (*db.TrackEmotions, error) {
			return nil, nil // track exists but emotions not computed yet
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/99/emotions", nil)
	w := httptest.NewRecorder()
	h.GetTrackEmotionsHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetTrackEmotionsHandler_DBError(t *testing.T) {
	store := &mockTrackStore{
		getEmotionsFunc: func(_ context.Context, _ int64) (*db.TrackEmotions, error) {
			return nil, errors.New("db error")
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/1/emotions", nil)
	w := httptest.NewRecorder()
	h.GetTrackEmotionsHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestGetTrackEmotionsHandler_InvalidID(t *testing.T) {
	h := newTestHandlers(&mockTrackStore{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/abc/emotions", nil)
	w := httptest.NewRecorder()
	h.GetTrackEmotionsHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetTrackEmotionsHandler_MissingID(t *testing.T) {
	h := newTestHandlers(&mockTrackStore{})

	// Path with nothing after /tracks/
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks//emotions", nil)
	w := httptest.NewRecorder()
	h.GetTrackEmotionsHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ── GetEmotionsDistributionHandler ───────────────────────────────────────────

func TestGetEmotionsDistributionHandler_Success(t *testing.T) {
	dist := []db.EmotionDistribution{
		{Dimension: "tension", Min: 0.1, Max: 0.9, Avg: 0.45, Count: 500},
		{Dimension: "euphoria", Min: 0.2, Max: 0.95, Avg: 0.60, Count: 500},
	}
	store := &mockTrackStore{
		getEmotionsDistributionFunc: func(_ context.Context) ([]db.EmotionDistribution, error) {
			return dist, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emotions/distribution", nil)
	w := httptest.NewRecorder()
	h.GetEmotionsDistributionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp["count"].(float64) != 2 {
		t.Errorf("count = %v, want 2", resp["count"])
	}
	if _, ok := resp["distribution"]; !ok {
		t.Error("response missing 'distribution' key")
	}
}

func TestGetEmotionsDistributionHandler_Empty(t *testing.T) {
	store := &mockTrackStore{
		getEmotionsDistributionFunc: func(_ context.Context) ([]db.EmotionDistribution, error) {
			return []db.EmotionDistribution{}, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emotions/distribution", nil)
	w := httptest.NewRecorder()
	h.GetEmotionsDistributionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetEmotionsDistributionHandler_DBError(t *testing.T) {
	store := &mockTrackStore{
		getEmotionsDistributionFunc: func(_ context.Context) ([]db.EmotionDistribution, error) {
			return nil, errors.New("connection lost")
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emotions/distribution", nil)
	w := httptest.NewRecorder()
	h.GetEmotionsDistributionHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
