package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/costuff/doyou-music-analytics/internal/db"
	"github.com/costuff/doyou-music-analytics/pkg/logger"
)

// TrackStore defines the database operations required by the API handlers.
type TrackStore interface {
	GetTrackStats(ctx context.Context) (map[string]int64, error)
	ListTracks(ctx context.Context, limit, offset int, status string) ([]db.TrackWithFeatures, int64, error)
	GetTrackByID(ctx context.Context, trackID int64) (*db.TrackWithFeatures, error)
	SearchTracks(ctx context.Context, query string, limit int) ([]db.TrackWithFeatures, error)
	GetFailedTracks(ctx context.Context, limit int) ([]db.TrackWithFeatures, error)
	GetRecentCompletedTracks(ctx context.Context, limit int) ([]db.TrackWithFeatures, error)
	GetEmotions(ctx context.Context, trackID int64) (*db.TrackEmotions, error)
	GetEmotionsDistribution(ctx context.Context) ([]db.EmotionDistribution, error)
	GetAllEmotions(ctx context.Context, limit, offset int) ([]db.TrackEmotions, int64, error)
	GetSimilarTracks(ctx context.Context, trackID int64, n int) ([]db.TrackSimilarityResult, error)
	GetSimilarityStats(ctx context.Context) (db.SimilarityStats, error)
	ComputeSimilarity(ctx context.Context, k int) (int, error)
}

type Handlers struct {
	recomputeInProgress atomic.Bool
	db      TrackStore
	log     *logger.Logger
	version string
}

func NewHandlers(database TrackStore, log *logger.Logger, version string) *Handlers {
	return &Handlers{db: database, log: log, version: version}
}

func (h *Handlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"version":   h.version,
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "doyou-music-analytics",
	})
}

func (h *Handlers) StatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := h.db.GetTrackStats(r.Context())
	if err != nil {
		h.log.Error(fmt.Sprintf("Failed to get stats: %v", err))
		JSONError(w, http.StatusInternalServerError, "Failed to retrieve statistics")
		return
	}
	JSON(w, http.StatusOK, map[string]interface{}{
		"total": stats["total"], "pending": stats["pending"],
		"analyzing": stats["analyzing"], "completed": stats["completed"], "failed": stats["failed"],
	})
}

func (h *Handlers) ListTracksHandler(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	status := r.URL.Query().Get("status")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}
	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	tracks, total, err := h.db.ListTracks(r.Context(), limit, offset, status)
	if err != nil {
		h.log.Error(fmt.Sprintf("Failed to list tracks: %v", err))
		JSONError(w, http.StatusInternalServerError, "Failed to retrieve tracks")
		return
	}
	JSON(w, http.StatusOK, map[string]interface{}{"tracks": tracks, "total": total, "limit": limit, "offset": offset})
}

func (h *Handlers) GetTrackHandler(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/tracks/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		JSONError(w, http.StatusBadRequest, "Track ID is required")
		return
	}
	trackID, err := strconv.ParseInt(pathParts[0], 10, 64)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "Invalid track ID")
		return
	}
	track, err := h.db.GetTrackByID(r.Context(), trackID)
	if err != nil {
		h.log.Error(fmt.Sprintf("Failed to get track %d: %v", trackID, err))
		JSONError(w, http.StatusNotFound, "Track not found")
		return
	}
	JSON(w, http.StatusOK, track)
}

func (h *Handlers) SearchTracksHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		JSONError(w, http.StatusBadRequest, "Search query 'q' is required")
		return
	}
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}
	tracks, err := h.db.SearchTracks(r.Context(), query, limit)
	if err != nil {
		h.log.Error(fmt.Sprintf("Failed to search tracks '%s': %v", query, err))
		JSONError(w, http.StatusInternalServerError, "Search failed")
		return
	}
	JSON(w, http.StatusOK, map[string]interface{}{"query": query, "tracks": tracks, "count": len(tracks), "limit": limit})
}

func (h *Handlers) FailedTracksHandler(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}
	tracks, err := h.db.GetFailedTracks(r.Context(), limit)
	if err != nil {
		h.log.Error(fmt.Sprintf("Failed to get failed tracks: %v", err))
		JSONError(w, http.StatusInternalServerError, "Failed to retrieve failed tracks")
		return
	}
	JSON(w, http.StatusOK, map[string]interface{}{"failed_tracks": tracks, "count": len(tracks), "limit": limit})
}

// GetTrackEmotionsHandler handles GET /api/v1/tracks/:id/emotions
func (h *Handlers) GetTrackEmotionsHandler(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/tracks/"), "/")
	// expect ["<id>", "emotions"]
	if len(pathParts) < 2 || pathParts[0] == "" {
		JSONError(w, http.StatusBadRequest, "Track ID is required")
		return
	}
	trackID, err := strconv.ParseInt(pathParts[0], 10, 64)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "Invalid track ID")
		return
	}
	emotions, err := h.db.GetEmotions(r.Context(), trackID)
	if err != nil {
		h.log.Error(fmt.Sprintf("Failed to get emotions for track %d: %v", trackID, err))
		JSONError(w, http.StatusInternalServerError, "Failed to retrieve emotions")
		return
	}
	if emotions == nil {
		JSONError(w, http.StatusNotFound, "Emotions not yet computed for this track")
		return
	}
	JSON(w, http.StatusOK, emotions)
}

// GetEmotionsDistributionHandler handles GET /api/v1/emotions/distribution
func (h *Handlers) GetEmotionsDistributionHandler(w http.ResponseWriter, r *http.Request) {
	dist, err := h.db.GetEmotionsDistribution(r.Context())
	if err != nil {
		h.log.Error(fmt.Sprintf("Failed to get emotions distribution: %v", err))
		JSONError(w, http.StatusInternalServerError, "Failed to retrieve emotions distribution")
		return
	}
	JSON(w, http.StatusOK, map[string]interface{}{"distribution": dist, "count": len(dist)})
}

// GetAllEmotionsHandler handles GET /api/v1/emotions — bulk fetch for the emotion graph.
func (h *Handlers) GetAllEmotionsHandler(w http.ResponseWriter, r *http.Request) {
	limit := 2000
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	emotions, total, err := h.db.GetAllEmotions(r.Context(), limit, offset)
	if err != nil {
		h.log.Error(fmt.Sprintf("Failed to get all emotions: %v", err))
		JSONError(w, http.StatusInternalServerError, "Failed to retrieve emotions")
		return
	}
	if emotions == nil {
		emotions = []db.TrackEmotions{}
	}
	JSON(w, http.StatusOK, map[string]interface{}{
		"emotions": emotions,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

func (h *Handlers) DetailedStatsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stats, err := h.db.GetTrackStats(ctx)
	if err != nil {
		h.log.Error(fmt.Sprintf("Failed to get stats: %v", err))
		JSONError(w, http.StatusInternalServerError, "Failed to retrieve statistics")
		return
	}
	recentTracks, err := h.db.GetRecentCompletedTracks(ctx, 100)
	if err != nil {
		h.log.Error(fmt.Sprintf("Failed to get recent tracks: %v", err))
	}
	var avgProcessingTime float64
	var tracksPerHour float64
	if len(recentTracks) > 1 {
		var totalDuration time.Duration
		for _, track := range recentTracks {
			if track.AnalyzedAt != nil && track.DownloadedAt != nil {
				duration := track.AnalyzedAt.Sub(*track.DownloadedAt)
				totalDuration += duration
			}
		}
		if totalDuration > 0 {
			avgProcessingTime = totalDuration.Seconds() / float64(len(recentTracks))
			if len(recentTracks) > 1 {
				firstTime := recentTracks[len(recentTracks)-1].AnalyzedAt
				lastTime := recentTracks[0].AnalyzedAt
				if firstTime != nil && lastTime != nil {
					timeSpan := lastTime.Sub(*firstTime).Hours()
					if timeSpan > 0 {
						tracksPerHour = float64(len(recentTracks)) / timeSpan
					}
				}
			}
		}
	}
	response := map[string]interface{}{
		"total": stats["total"], "pending": stats["pending"], "analyzing": stats["analyzing"],
		"completed": stats["completed"], "failed": stats["failed"],
		"avg_processing_time_sec": fmt.Sprintf("%.2f", avgProcessingTime),
		"tracks_per_hour":         fmt.Sprintf("%.2f", tracksPerHour),
		"completion_percentage":   "0.00",
	}
	if stats["total"] > 0 {
		completionPct := float64(stats["completed"]) / float64(stats["total"]) * 100
		response["completion_percentage"] = fmt.Sprintf("%.2f", completionPct)
	}
	JSON(w, http.StatusOK, response)
}
// GetSimilarTracksHandler handles GET /api/v1/tracks/:id/similar?n=10
// Returns the pre-computed top-n nearest neighbours for the given track.
func (h *Handlers) GetSimilarTracksHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/tracks/")
	idStr = strings.TrimSuffix(idStr, "/similar")
	trackID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || trackID <= 0 {
		h.log.Errorf("GetSimilarTracks: invalid track id %q", r.URL.Path)
		http.Error(w, "invalid track id", http.StatusBadRequest)
		return
	}

	n := 10
	if q := r.URL.Query().Get("n"); q != "" {
		if parsed, err := strconv.Atoi(q); err == nil && parsed > 0 {
			n = parsed
		}
	}

	results, err := h.db.GetSimilarTracks(r.Context(), trackID, n)
	if err != nil {
		h.log.Errorf("GetSimilarTracks: db error for track %d: %v", trackID, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if results == nil {
		results = []db.TrackSimilarityResult{}
	}
	JSON(w, http.StatusOK, map[string]interface{}{
		"track_id": trackID,
		"similar":  results,
		"count":    len(results),
	})
}

// RecomputeSimilarityHandler handles POST /api/v1/similarity/recompute
// Triggers a background similarity recomputation and returns 202 immediately.
// Returns 409 if a recomputation is already running.
func (h *Handlers) RecomputeSimilarityHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.recomputeInProgress.CompareAndSwap(false, true) {
		JSON(w, http.StatusConflict, map[string]string{"status": "already_running"})
		return
	}
	go func() {
		defer h.recomputeInProgress.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		n, err := h.db.ComputeSimilarity(ctx, 20)
		if err != nil {
			h.log.Errorf("RecomputeSimilarity: failed: %v", err)
			return
		}
		h.log.Infof("RecomputeSimilarity: done, %d pairs stored", n)
	}()
	JSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

// GetSimilarityStatsHandler handles GET /api/v1/similarity/stats
func (h *Handlers) GetSimilarityStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := h.db.GetSimilarityStats(r.Context())
	if err != nil {
		h.log.Errorf("GetSimilarityStats: db error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	stats2 := map[string]interface{}{
		"total_pairs":     stats.TotalPairs,
		"covered_tracks":  stats.CoveredTracks,
		"last_computed":   stats.LastComputed,
		"recomputing":     h.recomputeInProgress.Load(),
	}
	JSON(w, http.StatusOK, stats2)
}