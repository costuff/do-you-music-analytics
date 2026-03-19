package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/costuff/doyou-music-analytics/internal/analyzer"
	"github.com/costuff/doyou-music-analytics/internal/config"
	"github.com/costuff/doyou-music-analytics/internal/db"
	"github.com/costuff/doyou-music-analytics/pkg/logger"
)

func main() {
	// Initialize logger
	log := logger.New("info")

	version := getVersion()
	log.Info(fmt.Sprintf("🚀 DoYou Music Analytics Worker v%s", version))
	log.Info("=====================================")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to load configuration: %v", err))
	}

	log.Info("✅ Configuration loaded")
	log.Info(fmt.Sprintf("   Database: %s", cfg.MaskPassword()))
	log.Info(fmt.Sprintf("   Music Directory: %s", cfg.NASMountPath))
	log.Info(fmt.Sprintf("   Batch Size: %d", cfg.BatchSize))
	log.Info(fmt.Sprintf("   Max Workers: %d", cfg.MaxWorkers))

	// Connect to database
	ctx := context.Background()
	database, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to connect to database: %v", err))
	}
	defer database.Close()

	log.Info("✅ Connected to PostgreSQL")

	// Initialize analyzer
	analyzerTimeout, err := time.ParseDuration(cfg.AnalysisTimeout)
	if err != nil {
		log.Fatal(fmt.Sprintf("Invalid analysis timeout: %v", err))
	}

	anlz := analyzer.New(cfg.PythonPath, cfg.AnalysisScript, analyzerTimeout)
	if err := anlz.Validate(); err != nil {
		log.Fatal(fmt.Sprintf("Analyzer validation failed: %v", err))
	}

	log.Info("✅ Analyzer initialized")

	// Get initial statistics
	stats, err := database.GetTrackStats(ctx)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to get track statistics: %v", err))
	}

	log.Info("📊 Initial Track Statistics:")
	log.Info(fmt.Sprintf("   Total: %d", stats["total"]))
	log.Info(fmt.Sprintf("   Pending: %d", stats["pending"]))
	log.Info(fmt.Sprintf("   Completed: %d", stats["completed"]))
	log.Info(fmt.Sprintf("   Failed: %d", stats["failed"]))

	// Parse scan interval
	scanInterval, err := time.ParseDuration(cfg.ScanInterval)
	if err != nil {
		log.Fatal(fmt.Sprintf("Invalid scan interval: %v", err))
	}

	log.Info(fmt.Sprintf("🔍 Automatic scan interval: %v", scanInterval))

	// Setup graceful shutdown
	shutdownCtx, shutdownCancel := context.WithCancel(ctx)
	defer shutdownCancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start worker pool
	log.Info(fmt.Sprintf("🔄 Starting worker pool with %d workers...", cfg.MaxWorkers))
	log.Info("🔄 Worker will run continuously, scanning for new tracks and processing pending ones")

	var wg sync.WaitGroup
	var processed, failed atomic.Int64
	startTime := time.Now()

	// Start periodic cleanup if enabled
	if cfg.EnableCleanup {
		cleanupInterval, err := time.ParseDuration(cfg.CleanupInterval)
		if err != nil {
			log.Fatal(fmt.Sprintf("Invalid cleanup interval: %v", err))
		}
		log.Info(fmt.Sprintf("🧹 Cleanup enabled: %s missing files every %v", cfg.CleanupAction, cleanupInterval))

		wg.Add(1)
		go func() {
			defer wg.Done()
			cleanupTicker := time.NewTicker(cleanupInterval)
			defer cleanupTicker.Stop()

			// Run initial cleanup after 5 minutes
			time.Sleep(5 * time.Minute)
			log.Info("🧹 Running initial cleanup check...")
			stats, err := database.CleanupMissingTracks(shutdownCtx, cfg.NASMountPath, cfg.CleanupAction)
			if err != nil {
				log.Error(fmt.Sprintf("Cleanup failed: %v", err))
			} else if stats.MissingTracks > 0 {
				if cfg.CleanupAction == "mark" {
					log.Info(fmt.Sprintf("✅ Marked %d missing tracks", stats.MarkedMissing))
				} else {
					log.Info(fmt.Sprintf("✅ Deleted %d missing tracks", stats.Deleted))
				}
			} else {
				log.Info("✅ No missing tracks found")
			}

			for {
				select {
				case <-shutdownCtx.Done():
					log.Info("Cleanup shutting down")
					return
				case <-cleanupTicker.C:
					log.Info("🧹 Running cleanup check...")
					stats, err := database.CleanupMissingTracks(shutdownCtx, cfg.NASMountPath, cfg.CleanupAction)
					if err != nil {
						log.Error(fmt.Sprintf("Cleanup failed: %v", err))
					} else if stats.MissingTracks > 0 {
						if cfg.CleanupAction == "mark" {
							log.Info(fmt.Sprintf("✅ Marked %d missing tracks", stats.MarkedMissing))
						} else {
							log.Info(fmt.Sprintf("✅ Deleted %d missing tracks", stats.Deleted))
						}
					} else {
						log.Debug("No missing tracks found")
					}
				}
			}
		}()
	} else {
		log.Info("🧹 Cleanup disabled (set ENABLE_CLEANUP=false if intentional)")
	}

	// Start periodic scanner in background
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanTicker := time.NewTicker(scanInterval)
		defer scanTicker.Stop()

		// Run initial scan immediately
		scanMode := "full recursive"
		if cfg.ShallowScan {
			scanMode = "shallow (top-level only)"
		}
		log.Info(fmt.Sprintf("🔍 Running initial scan for new tracks [%s]...", scanMode))
		newTracks, err := database.ScanAndImportNewTracks(shutdownCtx, cfg.NASMountPath, cfg.ShallowScan)
		if err != nil {
			log.Error(fmt.Sprintf("Initial scan failed: %v", err))
		} else if newTracks > 0 {
			log.Info(fmt.Sprintf("✅ Imported %d new tracks", newTracks))
		} else {
			log.Info("✅ No new tracks found")
		}

		for {
			select {
			case <-shutdownCtx.Done():
				log.Info("Scanner shutting down")
				return
			case <-scanTicker.C:
				log.Info("🔍 Scanning for new tracks...")
				newTracks, err := database.ScanAndImportNewTracks(shutdownCtx, cfg.NASMountPath, cfg.ShallowScan)
				if err != nil {
					log.Error(fmt.Sprintf("Scan failed: %v", err))
				} else if newTracks > 0 {
					log.Info(fmt.Sprintf("✅ Imported %d new tracks", newTracks))
				} else {
					log.Debug("No new tracks found")
				}
			}
		}
	}()

	workerFunc := func(workerID int) {
		defer wg.Done()

		for {
			select {
			case <-shutdownCtx.Done():
				log.Info(fmt.Sprintf("Worker %d shutting down gracefully", workerID))
				return
			default:
				// Atomically claim next pending track (prevents race conditions)
				track, err := database.ClaimPendingTrack(shutdownCtx)
				if err != nil {
					log.Error(fmt.Sprintf("Worker %d: Failed to claim pending track: %v", workerID, err))
					time.Sleep(5 * time.Second)
					continue
				}

				if track == nil {
					// No pending tracks, sleep briefly and try again
					time.Sleep(10 * time.Second)
					continue
				}

				// Build full file path
				// If the track.FilePath is already absolute, use it as-is
				// Otherwise, join it with the NAS mount path
				filePath := track.FilePath
				if !filepath.IsAbs(track.FilePath) {
					filePath = filepath.Join(cfg.NASMountPath, track.FilePath)
				}

				// Analyze track
				log.Info(fmt.Sprintf("Worker %d: Analyzing [%d] %s", workerID, track.ID, filePath))

				features, err := anlz.AnalyzeFile(shutdownCtx, filePath)
				if err != nil {
					errMsg := err.Error()
					log.Error(fmt.Sprintf("Worker %d: Analysis failed for %s: %v", workerID, track.FilePath, err))
					_ = database.UpdateTrackStatus(shutdownCtx, track.ID, "failed", &errMsg)
					failed.Add(1)
					continue
				}

				// Convert analyzer features to database features
				dbFeatures := &db.AudioFeatures{
					TrackID:          track.ID,
					BPM:              features.BPM,
					Key:              features.Key,
					Mode:             features.Mode,
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
					// Raw spectral features
					SpectralCentroidMean: features.SpectralCentroidMean,
					SpectralCentroidStd:  features.SpectralCentroidStd,
					SpectralRolloffMean:  features.SpectralRolloffMean,
					ZeroCrossingRateMean: features.ZeroCrossingRateMean,
					RMSEnergyMean:        features.RMSEnergyMean,
					RMSEnergyStd:         features.RMSEnergyStd,
					// MFCCs 0–12
					MFCC0Mean:  features.MFCC0Mean,
					MFCC0Std:   features.MFCC0Std,
					MFCC1Mean:  features.MFCC1Mean,
					MFCC1Std:   features.MFCC1Std,
					MFCC2Mean:  features.MFCC2Mean,
					MFCC2Std:   features.MFCC2Std,
					MFCC3Mean:  features.MFCC3Mean,
					MFCC3Std:   features.MFCC3Std,
					MFCC4Mean:  features.MFCC4Mean,
					MFCC4Std:   features.MFCC4Std,
					MFCC5Mean:  features.MFCC5Mean,
					MFCC5Std:   features.MFCC5Std,
					MFCC6Mean:  features.MFCC6Mean,
					MFCC6Std:   features.MFCC6Std,
					MFCC7Mean:  features.MFCC7Mean,
					MFCC7Std:   features.MFCC7Std,
					MFCC8Mean:  features.MFCC8Mean,
					MFCC8Std:   features.MFCC8Std,
					MFCC9Mean:  features.MFCC9Mean,
					MFCC9Std:   features.MFCC9Std,
					MFCC10Mean: features.MFCC10Mean,
					MFCC10Std:  features.MFCC10Std,
					MFCC11Mean: features.MFCC11Mean,
					MFCC11Std:  features.MFCC11Std,
					MFCC12Mean: features.MFCC12Mean,
					MFCC12Std:  features.MFCC12Std,
					AnalysisVersion: "2.0",
				}

				// Save features to database
				if err := database.SaveAudioFeatures(shutdownCtx, dbFeatures); err != nil {
					errMsg := fmt.Sprintf("Failed to save features: %v", err)
					log.Error(fmt.Sprintf("Worker %d: %s", workerID, errMsg))
					_ = database.UpdateTrackStatus(shutdownCtx, track.ID, "failed", &errMsg)
					failed.Add(1)
					continue
				}

				// Update status to completed
				if err := database.UpdateTrackStatus(shutdownCtx, track.ID, "completed", nil); err != nil {
					log.Error(fmt.Sprintf("Worker %d: Failed to update completion status: %v", workerID, err))
					continue
				}

				processed.Add(1)

				// Log progress every 10 tracks
				if processed.Load()%10 == 0 {
					elapsed := time.Since(startTime)
					rate := float64(processed.Load()) / elapsed.Seconds() * 60 // per minute
					remaining := int(stats["pending"]) - int(processed.Load())
					eta := time.Duration(float64(remaining)/rate) * time.Minute

					log.Info(fmt.Sprintf("📊 Progress: %d/%d completed, %d failed | Rate: %.1f/min | ETA: %v",
						processed.Load(), stats["pending"], failed.Load(), rate, eta.Round(time.Minute)))
				}
			}
		}
	}

	// Start workers
	for i := 1; i <= cfg.MaxWorkers; i++ {
		wg.Add(1)
		go workerFunc(i)
	}

	// Wait for shutdown signal (workers run forever until signaled)
	sig := <-sigChan
	log.Info(fmt.Sprintf("⚠️  Received signal: %v", sig))
	log.Info("🛑 Initiating graceful shutdown...")
	shutdownCancel()

	// Wait for all workers with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("✅ All workers stopped gracefully")
	case <-time.After(30 * time.Second):
		log.Warn("⚠️  Timeout waiting for workers, forcing shutdown")
	}

	// Final statistics
	elapsed := time.Since(startTime)
	log.Info("📊 Final Statistics:")
	log.Info(fmt.Sprintf("   Processed: %d tracks", processed.Load()))
	log.Info(fmt.Sprintf("   Failed: %d tracks", failed.Load()))
	log.Info(fmt.Sprintf("   Duration: %v", elapsed.Round(time.Second)))
	if processed.Load() > 0 {
		avgTime := elapsed / time.Duration(processed.Load())
		log.Info(fmt.Sprintf("   Average: %v per track", avgTime.Round(time.Second)))
	}

	log.Info("👋 Worker shutdown complete")
}

func getVersion() string {
	version, err := os.ReadFile("VERSION")
	if err != nil {
		return "unknown"
	}
	return string(version)
}
