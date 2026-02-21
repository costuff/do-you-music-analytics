package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScanStats contains statistics from scanning operations
type ScanStats struct {
	NewTracks     int
	MissingTracks int
	MarkedMissing int
	Deleted       int
}

// ScanAndImportNewTracks scans the music directory for MP3 files and imports new ones
// Returns the number of newly imported tracks
func (db *DB) ScanAndImportNewTracks(ctx context.Context, musicDir string) (int, error) {
	// Get all existing file paths from database
	existingPaths := make(map[string]bool)
	rows, err := db.pool.Query(ctx, "SELECT file_path FROM tracks")
	if err != nil {
		return 0, fmt.Errorf("failed to query existing tracks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return 0, fmt.Errorf("failed to scan path: %w", err)
		}
		existingPaths[path] = true
	}

	// Scan directory for MP3 files
	var newFiles []string
	err = filepath.Walk(musicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mp3") {
			// Get relative path from music directory
			relPath, err := filepath.Rel(musicDir, path)
			if err != nil {
				return err
			}

			// Check if this file is already in database
			if !existingPaths[relPath] && !existingPaths[path] {
				newFiles = append(newFiles, path)
			}
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to scan directory: %w", err)
	}

	// Import new files
	imported := 0
	for _, filePath := range newFiles {
		fileName := filepath.Base(filePath)
		relPath, _ := filepath.Rel(musicDir, filePath)

		query := `
			INSERT INTO tracks (file_path, file_name, analysis_status, downloaded_at)
			VALUES ($1, $2, 'pending', NOW())
			ON CONFLICT (file_path) DO NOTHING
		`
		_, err := db.pool.Exec(ctx, query, relPath, fileName)
		if err != nil {
			return imported, fmt.Errorf("failed to insert track: %w", err)
		}
		imported++
	}

	return imported, nil
}

// CleanupMissingTracks checks for tracks in DB where files no longer exist
// Returns stats about what was cleaned up
func (db *DB) CleanupMissingTracks(ctx context.Context, musicDir string, action string) (*ScanStats, error) {
	stats := &ScanStats{}

	// Validate action
	if action != "mark" && action != "delete" {
		return nil, fmt.Errorf("invalid action: %s (must be 'mark' or 'delete')", action)
	}

	// Get all tracks from database that aren't already marked as missing
	query := `SELECT id, file_path FROM tracks WHERE analysis_status != 'missing'`
	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tracks: %w", err)
	}
	defer rows.Close()

	type trackRecord struct {
		id   int64
		path string
	}
	var tracks []trackRecord

	for rows.Next() {
		var t trackRecord
		if err := rows.Scan(&t.id, &t.path); err != nil {
			return nil, fmt.Errorf("failed to scan track: %w", err)
		}
		tracks = append(tracks, t)
	}

	// Check if each file exists
	for _, track := range tracks {
		// Try both relative and absolute paths
		fullPath := filepath.Join(musicDir, track.path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			// Also try the path as-is in case it's already absolute
			if _, err := os.Stat(track.path); os.IsNotExist(err) {
				stats.MissingTracks++

				if action == "mark" {
					// Mark as missing
					updateQuery := `
						UPDATE tracks 
						SET analysis_status = 'missing',
						    analysis_error = 'File no longer exists on filesystem'
						WHERE id = $1
					`
					_, err := db.pool.Exec(ctx, updateQuery, track.id)
					if err != nil {
						return stats, fmt.Errorf("failed to mark track %d as missing: %w", track.id, err)
					}
					stats.MarkedMissing++
				} else if action == "delete" {
					// Delete from database
					deleteQuery := `DELETE FROM tracks WHERE id = $1`
					_, err := db.pool.Exec(ctx, deleteQuery, track.id)
					if err != nil {
						return stats, fmt.Errorf("failed to delete track %d: %w", track.id, err)
					}
					stats.Deleted++
				}
			}
		}
	}

	return stats, nil
}
