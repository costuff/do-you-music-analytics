package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/costuff/doyou-music-analytics/internal/config"
	"github.com/costuff/doyou-music-analytics/internal/db"
	"github.com/costuff/doyou-music-analytics/pkg/logger"
)

func main() {
	// Initialize logger
	log := logger.New("info")
	log.Info("Starting DoYou Music Analytics API")

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

	mux := http.NewServeMux()
	
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","version":"%s","timestamp":"%s"}`, 
			getVersion(), time.Now().Format(time.RFC3339))
	})

	mux.HandleFunc("/api/v1/stats", func(w http.ResponseWriter, r *http.Request) {
		stats, err := database.GetTrackStats(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"total":%d,"pending":%d,"analyzing":%d,"completed":%d,"failed":%d}`,
			stats["total"], stats["pending"], stats["analyzing"], stats["completed"], stats["failed"])
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
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

func getVersion() string {
	version, err := os.ReadFile("VERSION")
	if err != nil {
		return "unknown"
	}
	return string(version)
}
