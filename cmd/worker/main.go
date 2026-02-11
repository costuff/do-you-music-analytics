package main

import (
	"fmt"
	"log"

	"github.com/costuff/doyou-music-analytics/internal/config"
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

	fmt.Println("✅ Configuration loaded successfully")
	fmt.Printf("   Database: %s\n", cfg.MaskPassword())
	fmt.Printf("   NAS Mount: %s\n", cfg.NASMountPath)
	fmt.Printf("   Batch Size: %d\n", cfg.BatchSize)
	fmt.Printf("   Max Workers: %d\n", cfg.MaxWorkers)
	fmt.Printf("   Python: %s\n", cfg.PythonPath)

	fmt.Println("\n✅ Phase 1 Complete: Project structure initialized")
	fmt.Println("\n📝 Next steps:")
	fmt.Println("   1. Set up PostgreSQL: docker-compose up -d postgres")
	fmt.Println("   2. Run migrations: (to be implemented)")
	fmt.Println("   3. Implement database connection")
	fmt.Println("   4. Implement analyzer")
	fmt.Println("   5. Implement worker loop")
	fmt.Println("\n📖 See docs/ANALYTICS_ARCHITECTURE.md for details")
}
