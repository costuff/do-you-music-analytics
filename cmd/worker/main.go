package main

import (
	"context"
	"fmt"
	"log"

	"github.com/costuff/doyou-music-analytics/internal/config"
	"github.com/costuff/doyou-music-analytics/internal/db"
)

const version = "0.1.0"

func main() {
	fmt.Printf("DoYou Music Analytics Worker v%s\n", version)
	fmt.Println("=====================================\n")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Println("✅ Configuration loaded")
	fmt.Printf("   Database: %s\n", cfg.MaskPassword())
	fmt.Printf("   NAS Mount: %s\n", cfg.NASMountPath)
	fmt.Printf("   Batch Size: %d\n", cfg.BatchSize)
	fmt.Printf("   Max Workers: %d\n\n", cfg.MaxWorkers)

	// Connect to database
	ctx := context.Background()
	database, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	fmt.Println("✅ Connected to PostgreSQL")

	// Get track statistics
	stats, err := database.GetTrackStats(ctx)
	if err != nil {
		log.Fatalf("Failed to get track statistics: %v", err)
	}

	fmt.Println("\n📊 Track Statistics:")
	fmt.Printf("   Total: %d\n", stats["total"])
	fmt.Printf("   Pending: %d\n", stats["pending"])
	fmt.Printf("   Completed: %d\n", stats["completed"])
	fmt.Printf("   Failed: %d\n", stats["failed"])

	// Get pending tracks count
	pending, err := database.GetPendingTracks(ctx, 5)
	if err != nil {
		log.Fatalf("Failed to get pending tracks: %v", err)
	}

	fmt.Printf("\n📋 Sample pending tracks: %d found\n", len(pending))

	fmt.Println("\n✅ Phase 2 Complete: Database integration working!")
	fmt.Println("\n📝 Next steps:")
	fmt.Println("   1. ✅ PostgreSQL connected")
	fmt.Println("   2. ✅ Migrations applied")
	fmt.Println("   3. ✅ Database operations working")
	fmt.Println("   4. Implement analyzer (Go ↔ Python bridge)")
	fmt.Println("   5. Implement worker loop (batch processing)")
	fmt.Println("\n📖 See docs/ANALYTICS_ARCHITECTURE.md for details")
}
