package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/costuff/doyou-music-analytics/internal/config"
	"github.com/jackc/pgx/v5"
)

const version = "0.1.0"

func main() {
	fmt.Printf("DoYou Music Analytics - Database Migration Tool v%s\n", version)
	fmt.Println("========================================================")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Printf("Database: %s\n\n", cfg.MaskPassword())

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	fmt.Println("✅ Connected to PostgreSQL")

	switch command {
	case "up":
		if err := migrateUp(ctx, conn); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	case "down":
		if err := migrateDown(ctx, conn); err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}
	case "status":
		if err := showStatus(ctx, conn); err != nil {
			log.Fatalf("Status check failed: %v", err)
		}
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: migrate <command>")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  up      - Apply all pending migrations")
	fmt.Println("  down    - Rollback the last migration")
	fmt.Println("  status  - Show migration status")
}

func migrateUp(ctx context.Context, conn *pgx.Conn) error {
	createTable := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT NOW()
		)`

	_, err := conn.Exec(ctx, createTable)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	files, err := filepath.Glob("migrations/*.up.sql")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No migration files found in migrations/")
		return nil
	}

	sort.Strings(files)
	fmt.Printf("Found %d migration file(s)\n\n", len(files))

	appliedCount := 0
	for _, file := range files {
		version := filepath.Base(file)
		version = strings.TrimSuffix(version, ".up.sql")

		var exists bool
		err := conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		if exists {
			fmt.Printf("⏩ Skipping %s (already applied)\n", version)
			continue
		}

		fmt.Printf("⬆️  Applying %s...", version)
		sql, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		_, err = tx.Exec(ctx, string(sql))
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to apply %s: %w", version, err)
		}

		_, err = tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to record migration: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		fmt.Println(" ✅")
		appliedCount++
		time.Sleep(100 * time.Millisecond)
	}

	if appliedCount == 0 {
		fmt.Println("✅ All migrations already applied")
	} else {
		fmt.Printf("\n✅ Successfully applied %d migration(s)\n", appliedCount)
	}

	return nil
}

func migrateDown(ctx context.Context, conn *pgx.Conn) error {
	query := `SELECT version FROM schema_migrations ORDER BY applied_at DESC LIMIT 1`

	var lastVersion string
	err := conn.QueryRow(ctx, query).Scan(&lastVersion)
	if err != nil {
		if err == pgx.ErrNoRows {
			fmt.Println("No migrations to roll back")
			return nil
		}
		return fmt.Errorf("failed to get last migration: %w", err)
	}

	downFile := fmt.Sprintf("migrations/%s.down.sql", lastVersion)
	if _, err := os.Stat(downFile); os.IsNotExist(err) {
		return fmt.Errorf("rollback file not found: %s", downFile)
	}

	fmt.Printf("⬇️  Rolling back %s...", lastVersion)

	sql, err := os.ReadFile(downFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", downFile, err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	_, err = tx.Exec(ctx, string(sql))
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to rollback %s: %w", lastVersion, err)
	}

	_, err = tx.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1", lastVersion)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Println(" ✅")
	fmt.Printf("\n✅ Successfully rolled back %s\n", lastVersion)

	return nil
}

func showStatus(ctx context.Context, conn *pgx.Conn) error {
	checkQuery := `SELECT EXISTS (
		SELECT FROM information_schema.tables 
		WHERE table_name = 'schema_migrations'
	)`

	var tableExists bool
	err := conn.QueryRow(ctx, checkQuery).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check migrations table: %w", err)
	}

	if !tableExists {
		fmt.Println("⚠️  No migrations have been applied yet")
		return nil
	}

	rows, err := conn.Query(ctx, "SELECT version, applied_at FROM schema_migrations ORDER BY applied_at")
	if err != nil {
		return fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	fmt.Println("Applied migrations:")
	fmt.Println("-------------------")

	count := 0
	for rows.Next() {
		var version string
		var appliedAt time.Time
		if err := rows.Scan(&version, &appliedAt); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}
		fmt.Printf("✅ %s (applied %s)\n", version, appliedAt.Format("2006-01-02 15:04:05"))
		count++
	}

	if count == 0 {
		fmt.Println("(none)")
	} else {
		fmt.Printf("\nTotal: %d migration(s) applied\n", count)
	}

	return nil
}
