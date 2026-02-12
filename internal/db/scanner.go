package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
