package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/costuff/doyou-music-analytics/internal/analyzer"
	"github.com/costuff/doyou-music-analytics/internal/config"
	"github.com/costuff/doyou-music-analytics/internal/db"
)

func main() {
	fmt.Println("DoYou Music Analytics - Analyzer Test")
	fmt.Println("======================================")

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/test-analyzer/main.go <mp3-file>")
		os.Exit(1)
	}

	audioFile := os.Args[1]
	if _, err := os.Stat(audioFile); os.IsNotExist(err) {
		log.Fatalf("File not found: %s", audioFile)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	timeout, _ := time.ParseDuration(cfg.AnalysisTimeout)
	a := analyzer.New(cfg.PythonPath, cfg.AnalysisScript, timeout)

	if err := a.Validate(); err != nil {
		log.Fatalf("Analyzer validation failed: %v", err)
	}

	ctx := context.Background()
	database, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		log.Fatalf("Database error: %v", err)
	}
	defer database.Close()

	fileName := filepath.Base(audioFile)
	trackID, err := database.CreateTrack(ctx, audioFile, fileName)
	if err != nil {
		existing, _ := database.GetTrackByPath(ctx, audioFile)
		if existing != nil {
			trackID = existing.ID
		} else {
			log.Fatalf("Track creation failed: %v", err)
		}
	}

	fmt.Printf("Track ID: %d\n", trackID)
	fmt.Println("Analyzing (this may take 30-60s)...")

	database.UpdateTrackStatus(ctx, trackID, "analyzing", nil)

	features, err := a.AnalyzeFile(ctx, audioFile)
	if err != nil {
		errMsg := err.Error()
		database.UpdateTrackStatus(ctx, trackID, "failed", &errMsg)
		log.Fatalf("Analysis failed: %v", err)
	}

	fmt.Println("\nResults:")
	fmt.Printf("Duration: %.2fs\n", features.DurationSeconds)
	if features.BPM != nil {
		fmt.Printf("BPM: %.1f\n", *features.BPM)
	}
	if features.Key != nil {
		fmt.Printf("Key: %s\n", *features.Key)
	}
	if features.Energy != nil {
		fmt.Printf("Energy: %.3f\n", *features.Energy)
	}

	dbFeatures := &db.AudioFeatures{
		TrackID:          trackID,
		BPM:              features.BPM,
		Key:              features.Key,
		DurationSeconds:  &features.DurationSeconds,
		Energy:           features.Energy,
		Danceability:     features.Danceability,
		Valence:          features.Valence,
		Acousticness:     features.Acousticness,
		Instrumentalness: features.Instrumentalness,
		Speechiness:      features.Speechiness,
		Liveness:         features.Liveness,
		Melodicness:      features.Melodicness,
		LoudnessDB:       features.LoudnessDB,
		AnalysisVersion:  "1.0.0",
	}

	if err := database.SaveAudioFeatures(ctx, dbFeatures); err != nil {
		log.Fatalf("Save failed: %v", err)
	}

	database.UpdateTrackStatus(ctx, trackID, "completed", nil)

	fmt.Println("\n✅ Analysis complete and saved to database!")
}
