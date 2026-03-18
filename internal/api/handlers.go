package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
}

type Handlers struct {
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
