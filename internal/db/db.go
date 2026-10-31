package db

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

func Connect(ctx context.Context, databaseURL string, maxConns int) (*DB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	config.MaxConns = int32(maxConns)
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{pool: pool}, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// RunMigrations applies all pending *.up.sql migrations embedded in migrationsFS.
// It is idempotent — already-applied migrations are skipped.
func (db *DB) RunMigrations(ctx context.Context, migrationsFS embed.FS) error {
	_, err := db.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT NOW()
		)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := migrationsFS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	applied := 0
	for _, filename := range files {
		version := strings.TrimSuffix(filename, ".up.sql")

		sql, err := migrationsFS.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", filename, err)
		}

		tx, err := db.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", version, err)
		}
		// Serialize migration runners and re-check inside the transaction so concurrent API
		// instances cannot both execute the same non-idempotent DDL.
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(78423957241001)"); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("lock migration runner for %s: %w", version, err)
		}
		var exists bool
		if err := tx.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version,
		).Scan(&exists); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if exists {
			if err := tx.Commit(ctx); err != nil {
				return fmt.Errorf("commit migration lock %s: %w", version, err)
			}
			continue
		}
		legacySchema, err := legacyMigrationPresent(ctx, tx, version)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("inspect legacy schema for %s: %w", version, err)
		}
		if legacySchema {
			if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
				tx.Rollback(ctx)
				return fmt.Errorf("adopt legacy migration %s: %w", version, err)
			}
			if err := tx.Commit(ctx); err != nil {
				return fmt.Errorf("commit legacy migration %s: %w", version, err)
			}
			continue
		}
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx) //nolint:errcheck // rollback on error path
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			_ = tx.Rollback(ctx) //nolint:errcheck // rollback on error path
			return fmt.Errorf("record migration %s: %w", version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", version, err)
		}
		applied++
	}
	return nil
}

// legacyMigrationPresent recognizes schema objects created by the historical
// /docker-entrypoint-initdb.d bootstrap (which did not populate schema_migrations).
func legacyMigrationPresent(ctx context.Context, tx pgx.Tx, version string) (bool, error) {
	var query string
	switch version {
	case "001_initial_schema":
		query = `SELECT to_regclass('public.tracks') IS NOT NULL AND to_regclass('public.audio_features') IS NOT NULL`
	case "002_add_missing_status":
		query = `SELECT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'tracks'::regclass AND conname = 'valid_analysis_status' AND pg_get_constraintdef(oid) LIKE '%missing%')`
	case "003_add_raw_features":
		query = `SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='audio_features' AND column_name='mfcc_0_mean')`
	case "004_add_emotions":
		query = `SELECT to_regclass('public.track_emotions') IS NOT NULL`
	case "005_add_similarity":
		query = `SELECT to_regclass('public.track_similarity') IS NOT NULL`
	case "008_analysis_retries":
		query = `SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='tracks' AND column_name='analysis_attempts')`
	case "009_analysis_duration":
		query = `SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='tracks' AND column_name='analysis_duration_seconds')`
	default:
		return false, nil
	}
	var present bool
	if err := tx.QueryRow(ctx, query).Scan(&present); err != nil {
		return false, err
	}
	return present, nil
}
