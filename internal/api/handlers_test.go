package api

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/costuff/doyou-music-analytics/internal/db"
	"github.com/costuff/doyou-music-analytics/pkg/logger"
)

// mockTrackStore implements TrackStore for testing.
type mockTrackStore struct {
	getTrackStatsFunc            func(ctx context.Context) (map[string]int64, error)
	listTracksFunc               func(ctx context.Context, limit, offset int, status string) ([]db.TrackWithFeatures, int64, error)
	getTrackByIDFunc             func(ctx context.Context, trackID int64) (*db.TrackWithFeatures, error)
	searchTracksFunc             func(ctx context.Context, query string, limit int) ([]db.TrackWithFeatures, error)
	getFailedTracksFunc          func(ctx context.Context, limit int) ([]db.TrackWithFeatures, error)
	getRecentCompletedTracksFunc func(ctx context.Context, limit int) ([]db.TrackWithFeatures, error)
	getEmotionsFunc              func(ctx context.Context, trackID int64) (*db.TrackEmotions, error)
	getEmotionsDistributionFunc  func(ctx context.Context) ([]db.EmotionDistribution, error)
	getAllEmotionsFunc           func(ctx context.Context, limit, offset int) ([]db.TrackEmotions, int64, error)
	getSimilarTracksFunc         func(ctx context.Context, trackID int64, n int) ([]db.TrackSimilarityResult, error)
	getSimilarityStatsFunc       func(ctx context.Context) (db.SimilarityStats, error)
	computeSimilarityFunc        func(ctx context.Context, k int) (int, error)
	resetTracksForReanalysisFunc func(ctx context.Context, statusFilter string, dryRun bool) (int64, error)
	getAllProjectionsFunc        func(ctx context.Context, limit int) ([]db.TrackProjection, error)
	computeAllProjectionsFunc    func(ctx context.Context) (int, error)
	getTaxonomyTreeFunc          func(ctx context.Context) ([]*db.TaxonomyNode, error)
	addTaxonomyNodeFunc          func(ctx context.Context, parentID *int, name, description, source string) (*db.TaxonomyNode, error)
	getTrackLabelFunc            func(ctx context.Context, trackID int64) (*db.TrackLabel, error)
	setTrackLabelFunc            func(ctx context.Context, trackID int64, labelPath []string, curatedBy, notes string) (*db.TrackLabel, error)
	getLabelsByPathFunc          func(ctx context.Context, path []string) ([]db.TrackLabel, error)
	autoLabelTrackFunc           func(ctx context.Context, trackID int64) (*db.TrackLabel, error)
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
func (m *mockTrackStore) GetAllEmotions(ctx context.Context, limit, offset int) ([]db.TrackEmotions, int64, error) {
	if m.getAllEmotionsFunc != nil {
		return m.getAllEmotionsFunc(ctx, limit, offset)
	}
	return []db.TrackEmotions{}, 0, nil
}

func (m *mockTrackStore) GetSimilarTracks(ctx context.Context, trackID int64, n int) ([]db.TrackSimilarityResult, error) {
	if m.getSimilarTracksFunc != nil {
		return m.getSimilarTracksFunc(ctx, trackID, n)
	}
	return []db.TrackSimilarityResult{}, nil
}

func (m *mockTrackStore) GetSimilarityStats(ctx context.Context) (db.SimilarityStats, error) {
	if m.getSimilarityStatsFunc != nil {
		return m.getSimilarityStatsFunc(ctx)
	}
	return db.SimilarityStats{}, nil
}

func (m *mockTrackStore) ComputeSimilarity(ctx context.Context, k int) (int, error) {
	if m.computeSimilarityFunc != nil {
		return m.computeSimilarityFunc(ctx, k)
	}
	return 0, nil
}

func (m *mockTrackStore) ResetTracksForReanalysis(ctx context.Context, statusFilter string, dryRun bool) (int64, error) {
	if m.resetTracksForReanalysisFunc != nil {
		return m.resetTracksForReanalysisFunc(ctx, statusFilter, dryRun)
	}
	return 0, nil
}
func (m *mockTrackStore) GetAllProjections(ctx context.Context, limit int) ([]db.TrackProjection, error) {
	if m.getAllProjectionsFunc != nil {
		return m.getAllProjectionsFunc(ctx, limit)
	}
	return []db.TrackProjection{}, nil
}
func (m *mockTrackStore) ComputeAndSaveAllProjections(ctx context.Context) (int, error) {
	if m.computeAllProjectionsFunc != nil {
		return m.computeAllProjectionsFunc(ctx)
	}
	return 0, nil
}
func (m *mockTrackStore) GetTaxonomyTree(ctx context.Context) ([]*db.TaxonomyNode, error) {
	if m.getTaxonomyTreeFunc != nil {
		return m.getTaxonomyTreeFunc(ctx)
	}
	return []*db.TaxonomyNode{}, nil
}
func (m *mockTrackStore) AddTaxonomyNode(ctx context.Context, parentID *int, name, description, source string) (*db.TaxonomyNode, error) {
	if m.addTaxonomyNodeFunc != nil {
		return m.addTaxonomyNodeFunc(ctx, parentID, name, description, source)
	}
	return &db.TaxonomyNode{Name: name, Source: source}, nil
}
func (m *mockTrackStore) GetTrackLabel(ctx context.Context, trackID int64) (*db.TrackLabel, error) {
	if m.getTrackLabelFunc != nil {
		return m.getTrackLabelFunc(ctx, trackID)
	}
	return nil, errors.New("not found")
}
func (m *mockTrackStore) SetTrackLabel(ctx context.Context, trackID int64, labelPath []string, curatedBy, notes string) (*db.TrackLabel, error) {
	if m.setTrackLabelFunc != nil {
		return m.setTrackLabelFunc(ctx, trackID, labelPath, curatedBy, notes)
	}
	return &db.TrackLabel{TrackID: trackID, LabelPath: labelPath}, nil
}
func (m *mockTrackStore) GetLabelsByPath(ctx context.Context, path []string) ([]db.TrackLabel, error) {
	if m.getLabelsByPathFunc != nil {
		return m.getLabelsByPathFunc(ctx, path)
	}
	return []db.TrackLabel{}, nil
}
func (m *mockTrackStore) AutoLabelTrack(ctx context.Context, trackID int64) (*db.TrackLabel, error) {
	if m.autoLabelTrackFunc != nil {
		return m.autoLabelTrackFunc(ctx, trackID)
	}
	return &db.TrackLabel{TrackID: trackID, AutoLabelPath: []string{"serenity"}}, nil
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

// ── GetAllEmotionsHandler ──────────────────────────────────────────────────────

func TestGetAllEmotionsHandler_Success(t *testing.T) {
	now := time.Now()
	store := &mockTrackStore{
		getAllEmotionsFunc: func(_ context.Context, limit, offset int) ([]db.TrackEmotions, int64, error) {
			return []db.TrackEmotions{
				{TrackID: 1, Tension: 0.44, Melancholy: 0.50, Euphoria: 0.54, Aggression: 0.50,
					Serenity: 0.46, Nostalgia: 0.39, Sensuality: 0.39, Power: 0.47, ComputedAt: now},
				{TrackID: 2, Tension: 0.40, Melancholy: 0.39, Euphoria: 0.64, Aggression: 0.55,
					Serenity: 0.38, Nostalgia: 0.28, Sensuality: 0.50, Power: 0.55, ComputedAt: now},
			}, 2, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emotions", nil)
	w := httptest.NewRecorder()
	h.GetAllEmotionsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp["total"].(float64) != 2 {
		t.Errorf("total = %v, want 2", resp["total"])
	}
	emotions, ok := resp["emotions"].([]interface{})
	if !ok {
		t.Fatal("emotions field missing or wrong type")
	}
	if len(emotions) != 2 {
		t.Errorf("len(emotions) = %d, want 2", len(emotions))
	}
	// Verify response envelope fields
	for _, field := range []string{"emotions", "total", "limit", "offset"} {
		if _, ok := resp[field]; !ok {
			t.Errorf("response missing field: %s", field)
		}
	}
}

func TestGetAllEmotionsHandler_Empty(t *testing.T) {
	store := &mockTrackStore{
		getAllEmotionsFunc: func(_ context.Context, limit, offset int) ([]db.TrackEmotions, int64, error) {
			return []db.TrackEmotions{}, 0, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emotions", nil)
	w := httptest.NewRecorder()
	h.GetAllEmotionsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp["total"].(float64) != 0 {
		t.Errorf("total = %v, want 0", resp["total"])
	}
	// emotions should be [] not null
	emotions, ok := resp["emotions"].([]interface{})
	if !ok {
		t.Fatal("emotions should be an array even when empty")
	}
	if len(emotions) != 0 {
		t.Errorf("expected empty array, got %d elements", len(emotions))
	}
}

func TestGetAllEmotionsHandler_PaginationDefaults(t *testing.T) {
	var capturedLimit, capturedOffset int
	store := &mockTrackStore{
		getAllEmotionsFunc: func(_ context.Context, limit, offset int) ([]db.TrackEmotions, int64, error) {
			capturedLimit = limit
			capturedOffset = offset
			return []db.TrackEmotions{}, 0, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emotions", nil)
	w := httptest.NewRecorder()
	h.GetAllEmotionsHandler(w, req)

	if capturedLimit != 2000 {
		t.Errorf("default limit = %d, want 2000", capturedLimit)
	}
	if capturedOffset != 0 {
		t.Errorf("default offset = %d, want 0", capturedOffset)
	}
}

func TestGetAllEmotionsHandler_PaginationParams(t *testing.T) {
	var capturedLimit, capturedOffset int
	store := &mockTrackStore{
		getAllEmotionsFunc: func(_ context.Context, limit, offset int) ([]db.TrackEmotions, int64, error) {
			capturedLimit = limit
			capturedOffset = offset
			return []db.TrackEmotions{}, 0, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emotions?limit=50&offset=100", nil)
	w := httptest.NewRecorder()
	h.GetAllEmotionsHandler(w, req)

	if capturedLimit != 50 {
		t.Errorf("limit = %d, want 50", capturedLimit)
	}
	if capturedOffset != 100 {
		t.Errorf("offset = %d, want 100", capturedOffset)
	}
}

func TestGetAllEmotionsHandler_DBError(t *testing.T) {
	store := &mockTrackStore{
		getAllEmotionsFunc: func(_ context.Context, limit, offset int) ([]db.TrackEmotions, int64, error) {
			return nil, 0, errors.New("db unavailable")
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emotions", nil)
	w := httptest.NewRecorder()
	h.GetAllEmotionsHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// ---- GetSimilarTracksHandler ----

func TestGetSimilarTracksHandler_Success(t *testing.T) {
	store := &mockTrackStore{
		getSimilarTracksFunc: func(_ context.Context, trackID int64, n int) ([]db.TrackSimilarityResult, error) {
			return []db.TrackSimilarityResult{
				{TrackID: 2, Score: 0.95},
				{TrackID: 3, Score: 0.87},
			}, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/1/similar", nil)
	req = req // path is parsed by handler from URL
	w := httptest.NewRecorder()
	h.GetSimilarTracksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	similar, ok := resp["similar"].([]interface{})
	if !ok {
		t.Fatal("response.similar is missing or not an array")
	}
	if len(similar) != 2 {
		t.Errorf("expected 2 results, got %d", len(similar))
	}
}

func TestGetSimilarTracksHandler_InvalidID(t *testing.T) {
	h := newTestHandlers(&mockTrackStore{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/notanumber/similar", nil)
	w := httptest.NewRecorder()
	h.GetSimilarTracksHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetSimilarTracksHandler_DBError(t *testing.T) {
	store := &mockTrackStore{
		getSimilarTracksFunc: func(_ context.Context, _ int64, _ int) ([]db.TrackSimilarityResult, error) {
			return nil, errors.New("db gone")
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/5/similar", nil)
	w := httptest.NewRecorder()
	h.GetSimilarTracksHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// ---- RecomputeSimilarityHandler ----

func TestRecomputeSimilarityHandler_Success(t *testing.T) {
	store := &mockTrackStore{
		computeSimilarityFunc: func(_ context.Context, _ int) (int, error) { return 100, nil },
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/similarity/recompute", nil)
	w := httptest.NewRecorder()
	h.RecomputeSimilarityHandler(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}
}

func TestRecomputeSimilarityHandler_WrongMethod(t *testing.T) {
	h := newTestHandlers(&mockTrackStore{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/similarity/recompute", nil)
	w := httptest.NewRecorder()
	h.RecomputeSimilarityHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestRecomputeSimilarityHandler_AlreadyRunning(t *testing.T) {
	h := newTestHandlers(&mockTrackStore{})
	h.recomputeInProgress.Store(true) // simulate in-progress

	req := httptest.NewRequest(http.MethodPost, "/api/v1/similarity/recompute", nil)
	w := httptest.NewRecorder()
	h.RecomputeSimilarityHandler(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

// ---- GetSimilarityStatsHandler ----

func TestGetSimilarityStatsHandler_Success(t *testing.T) {
	store := &mockTrackStore{
		getSimilarityStatsFunc: func(_ context.Context) (db.SimilarityStats, error) {
			return db.SimilarityStats{TotalPairs: 320, CoveredTracks: 16}, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/similarity/stats", nil)
	w := httptest.NewRecorder()
	h.GetSimilarityStatsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["total_pairs"] == nil {
		t.Error("response missing total_pairs")
	}
}

func TestGetSimilarityStatsHandler_DBError(t *testing.T) {
	store := &mockTrackStore{
		getSimilarityStatsFunc: func(_ context.Context) (db.SimilarityStats, error) {
			return db.SimilarityStats{}, errors.New("db error")
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/similarity/stats", nil)
	w := httptest.NewRecorder()
	h.GetSimilarityStatsHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// ---- ExportTracksHandler ----

func makeTrackWithFeatures() db.TrackWithFeatures {
	bpm := 128.5
	key := "C"
	mode := "Major"
	dur := 240.0
	loud := -8.5
	energy := 0.7
	dance := 0.8
	valence := 0.6
	acou := 0.1
	instru := 0.05
	speech := 0.03
	live := 0.12
	melo := 0.55
	return db.TrackWithFeatures{
		ID:             1,
		FileName:       "track.mp3",
		FilePath:       "/music/track.mp3",
		AnalysisStatus: "completed",
		Features: &db.AudioFeatures{
			BPM: &bpm, Key: &key, Mode: &mode,
			DurationSeconds: &dur, LoudnessDB: &loud,
			Energy: &energy, Danceability: &dance, Valence: &valence,
			Acousticness: &acou, Instrumentalness: &instru,
			Speechiness: &speech, Liveness: &live, Melodicness: &melo,
		},
	}
}

func TestExportTracksHandler_JSON(t *testing.T) {
	track := makeTrackWithFeatures()
	store := &mockTrackStore{
		listTracksFunc: func(_ context.Context, limit, offset int, status string) ([]db.TrackWithFeatures, int64, error) {
			return []db.TrackWithFeatures{track}, 1, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/export?format=json", nil)
	w := httptest.NewRecorder()
	h.ExportTracksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	tracks, ok := result["tracks"].([]interface{})
	if !ok || len(tracks) != 1 {
		t.Errorf("expected 1 track in response, got %v", result["tracks"])
	}
}

func TestExportTracksHandler_CSV(t *testing.T) {
	track := makeTrackWithFeatures()
	store := &mockTrackStore{
		listTracksFunc: func(_ context.Context, limit, offset int, status string) ([]db.TrackWithFeatures, int64, error) {
			return []db.TrackWithFeatures{track}, 1, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/export?format=csv&status=completed", nil)
	w := httptest.NewRecorder()
	h.ExportTracksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/csv" {
		t.Errorf("expected text/csv, got %q", ct)
	}
	r := csv.NewReader(bytes.NewReader(w.Body.Bytes()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v", err)
	}
	if len(records) < 2 {
		t.Fatalf("expected header + 1 data row, got %d rows", len(records))
	}
	if records[0][0] != "id" {
		t.Errorf("expected first column header 'id', got %q", records[0][0])
	}
}

func TestExportTracksHandler_CSVNoFeatures(t *testing.T) {
	track := db.TrackWithFeatures{
		ID:             2,
		FileName:       "bare.mp3",
		FilePath:       "/music/bare.mp3",
		AnalysisStatus: "pending",
		Features:       nil,
	}
	store := &mockTrackStore{
		listTracksFunc: func(_ context.Context, limit, offset int, status string) ([]db.TrackWithFeatures, int64, error) {
			return []db.TrackWithFeatures{track}, 1, nil
		},
	}
	h := newTestHandlers(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/export?format=csv", nil)
	w := httptest.NewRecorder()
	h.ExportTracksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	r := csv.NewReader(bytes.NewReader(w.Body.Bytes()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v", err)
	}
	// Header + 1 track row with empty feature columns
	if len(records) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(records))
	}
}

func TestExportTracksHandler_InvalidFormat(t *testing.T) {
	h := newTestHandlers(&mockTrackStore{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/export?format=xml", nil)
	w := httptest.NewRecorder()
	h.ExportTracksHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestExportTracksHandler_InvalidLimit(t *testing.T) {
	h := newTestHandlers(&mockTrackStore{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/export?limit=bad", nil)
	w := httptest.NewRecorder()
	h.ExportTracksHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestExportTracksHandler_WrongMethod(t *testing.T) {
	h := newTestHandlers(&mockTrackStore{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tracks/export", nil)
	w := httptest.NewRecorder()
	h.ExportTracksHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestExportTracksHandler_DBError(t *testing.T) {
	store := &mockTrackStore{
		listTracksFunc: func(_ context.Context, limit, offset int, status string) ([]db.TrackWithFeatures, int64, error) {
			return nil, 0, errors.New("db down")
		},
	}
	h := newTestHandlers(store)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/export", nil)
	w := httptest.NewRecorder()
	h.ExportTracksHandler(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestExportTracksHandler_LimitClamped(t *testing.T) {
	// Limit > 10000 should be clamped silently to 10000
	store := &mockTrackStore{
		listTracksFunc: func(_ context.Context, limit, offset int, status string) ([]db.TrackWithFeatures, int64, error) {
			if limit > 10000 {
				t.Errorf("limit %d exceeded max 10000", limit)
			}
			return nil, 0, nil
		},
	}
	h := newTestHandlers(store)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/export?limit=99999", nil)
	w := httptest.NewRecorder()
	h.ExportTracksHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// ---- ReanalyzeTracksHandler ----

func TestReanalyzeTracksHandler_DryRun(t *testing.T) {
	store := &mockTrackStore{
		resetTracksForReanalysisFunc: func(_ context.Context, statusFilter string, dryRun bool) (int64, error) {
			if statusFilter != "failed" {
				t.Errorf("expected statusFilter=failed, got %q", statusFilter)
			}
			if !dryRun {
				t.Error("expected dryRun=true")
			}
			return 42, nil
		},
	}
	h := newTestHandlers(store)

	body := `{"status_filter":"failed","dry_run":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tracks/reanalyze", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ReanalyzeTracksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["count"] != float64(42) {
		t.Errorf("expected count=42, got %v", result["count"])
	}
	msg, _ := result["message"].(string)
	if !strings.Contains(msg, "dry run") {
		t.Errorf("expected dry run message, got %q", msg)
	}
}

func TestReanalyzeTracksHandler_Live(t *testing.T) {
	store := &mockTrackStore{
		resetTracksForReanalysisFunc: func(_ context.Context, statusFilter string, dryRun bool) (int64, error) {
			return 100, nil
		},
	}
	h := newTestHandlers(store)

	body := `{"status_filter":"all"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tracks/reanalyze", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ReanalyzeTracksHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestReanalyzeTracksHandler_InvalidFilter(t *testing.T) {
	h := newTestHandlers(&mockTrackStore{})
	body := `{"status_filter":"unknown"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tracks/reanalyze", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ReanalyzeTracksHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestReanalyzeTracksHandler_WrongMethod(t *testing.T) {
	h := newTestHandlers(&mockTrackStore{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tracks/reanalyze", nil)
	w := httptest.NewRecorder()
	h.ReanalyzeTracksHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestReanalyzeTracksHandler_InvalidBody(t *testing.T) {
	h := newTestHandlers(&mockTrackStore{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tracks/reanalyze", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ReanalyzeTracksHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestReanalyzeTracksHandler_DBError(t *testing.T) {
	store := &mockTrackStore{
		resetTracksForReanalysisFunc: func(_ context.Context, statusFilter string, dryRun bool) (int64, error) {
			return 0, errors.New("db error")
		},
	}
	h := newTestHandlers(store)
	body := `{"status_filter":"failed"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tracks/reanalyze", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ReanalyzeTracksHandler(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
