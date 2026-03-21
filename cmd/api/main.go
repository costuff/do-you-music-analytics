package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/costuff/doyou-music-analytics/internal/api"
	"github.com/costuff/doyou-music-analytics/internal/config"
	"github.com/costuff/doyou-music-analytics/internal/db"
	"github.com/costuff/doyou-music-analytics/migrations"
	"github.com/costuff/doyou-music-analytics/pkg/logger"
)

func main() {
	// Initialize logger
	log := logger.New("info")
	version := getVersion()
	log.Info(fmt.Sprintf("Starting DoYou Music Analytics API v%s", version))

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to load config: %v", err))
	}

	ctx := context.Background()
	database, err := db.Connect(ctx, cfg.DatabaseURL, 20)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to connect to database: %v", err))
	}
	defer database.Close()

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := database.Ping(pingCtx); err != nil {
		log.Fatal(fmt.Sprintf("Failed to ping database: %v", err))
	}
	log.Info("Database connection established")

	// Auto-apply pending migrations (idempotent — safe to run on every startup)
	if err := database.RunMigrations(ctx, migrations.FS); err != nil {
		log.Fatal(fmt.Sprintf("Failed to run migrations: %v", err))
	}
	log.Info("Migrations up to date")

	// Create handlers
	handlers := api.NewHandlers(database, log, version)

	// Setup routes
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", handlers.HealthHandler)

	// API v1 routes
	mux.HandleFunc("/api/v1/stats", handlers.StatsHandler)
	mux.HandleFunc("/api/v1/stats/detailed", handlers.DetailedStatsHandler)
	mux.HandleFunc("/api/v1/tracks", handlers.ListTracksHandler)
	mux.HandleFunc("/api/v1/tracks/", routeTrackByID(handlers))
	mux.HandleFunc("/api/v1/tracks/search", handlers.SearchTracksHandler)
	mux.HandleFunc("/api/v1/tracks/failed", handlers.FailedTracksHandler)
	mux.HandleFunc("/api/v1/tracks/export", handlers.ExportTracksHandler)
	mux.HandleFunc("/api/v1/tracks/reanalyze", handlers.ReanalyzeTracksHandler)
	mux.HandleFunc("/api/v1/emotions", handlers.GetAllEmotionsHandler)
	mux.HandleFunc("/api/v1/emotions/distribution", handlers.GetEmotionsDistributionHandler)
	mux.HandleFunc("/api/v1/similarity/recompute", handlers.RecomputeSimilarityHandler)
	mux.HandleFunc("/api/v1/similarity/stats", handlers.GetSimilarityStatsHandler)
	// Projections
	mux.HandleFunc("/api/v1/projections", handlers.GetProjectionsHandler)
	mux.HandleFunc("/api/v1/projections/recompute", handlers.RecomputeProjectionsHandler)
	// Taxonomy + curation
	mux.HandleFunc("/api/v1/taxonomy", handlers.GetTaxonomyHandler)
	mux.HandleFunc("/api/v1/taxonomy/nodes", handlers.AddTaxonomyNodeHandler)
	mux.HandleFunc("/api/v1/labels", handlers.GetLabelsByPathHandler)

	// Apply middleware
	handler := api.LoggingMiddleware(log)(api.CORSMiddleware(mux))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info(fmt.Sprintf("API server listening on port %s", port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(fmt.Sprintf("Server failed: %v", err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error(fmt.Sprintf("Server forced to shutdown: %v", err))
	}

	log.Info("Server stopped")
}

// routeTrackByID handles:
//
//	/api/v1/tracks/:id          → GetTrackHandler
//	/api/v1/tracks/:id/emotions → GetTrackEmotionsHandler
//	/api/v1/tracks/:id/similar  → GetSimilarTracksHandler
func routeTrackByID(handlers *api.Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if this is /api/v1/tracks/:id or /api/v1/tracks/search or /api/v1/tracks/failed
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/tracks/")

		// If path is empty or has query params, it's the list endpoint
		if path == "" || r.URL.RawQuery != "" {
			handlers.ListTracksHandler(w, r)
			return
		}

		// If path is "search", "failed", "export", or "reanalyze", those are handled by specific routes
		if path == "search" || path == "failed" || path == "export" || path == "reanalyze" {
			http.NotFound(w, r)
			return
		}

		// Sub-resources: /api/v1/tracks/:id/<sub>
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 2 {
			switch parts[1] {
			case "emotions":
				handlers.GetTrackEmotionsHandler(w, r)
				return
			case "similar":
				handlers.GetSimilarTracksHandler(w, r)
				return
			case "label":
				if r.Method == http.MethodPost {
					handlers.SetTrackLabelHandler(w, r)
				} else {
					handlers.GetTrackLabelHandler(w, r)
				}
				return
			}
		}

		// Otherwise it's a bare track ID
		handlers.GetTrackHandler(w, r)
	}
}

func getVersion() string {
	version, err := os.ReadFile("VERSION")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(version))
}
