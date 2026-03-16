package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// ScanStats contains statistics from scanning operations
type ScanStats struct {
	NewTracks     int
	MissingTracks int
	MarkedMissing int
	Deleted       int
}

// scanMP3Files returns all MP3 files in musicDir.
// When shallow is true only the top-level directory is read — fast over SMB/NFS mounts.
// When shallow is false a full recursive walk is performed for local/production use.
// Hidden files (prefixed with "." or "._") are always skipped.
func scanMP3Files(musicDir string, shallow bool) ([]string, error) {
	if shallow {
		entries, err := os.ReadDir(musicDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}
		var files []string
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "._") {
				continue
			}
			if strings.HasSuffix(strings.ToLower(name), ".mp3") {
				files = append(files, filepath.Join(musicDir, name))
			}
		}
		return files, nil
	}
	var files []string
	err := filepath.Walk(musicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) || errors.Is(err, syscall.EPERM) {
				if info != nil && info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			return err
		}
		name := info.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "._") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(name), ".mp3") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// ScanAndImportNewTracks scans the music directory for MP3 files and imports new ones.
// When shallowScan is true, only the top-level directory is read (no recursion) — use this
// over slow network mounts (SMB/NFS). When false, a full recursive walk is performed.
// Returns the number of newly imported tracks.
func (db *DB) ScanAndImportNewTracks(ctx context.Context, musicDir string, shallowScan bool) (int, error) {
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

	allFiles, err := scanMP3Files(musicDir, shallowScan)
	if err != nil {
		return 0, err
	}

	// Filter out files that already exist in the database
	var newFiles []string
	for _, filePath := range allFiles {
		relPath, _ := filepath.Rel(musicDir, filePath)
		if !existingPaths[relPath] && !existingPaths[filePath] && !existingPaths[filepath.Base(filePath)] {
			newFiles = append(newFiles, filePath)
		}
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
		// The primary path to check: join music dir with the stored relative path
		fullPath := filepath.Join(musicDir, track.path)
		_, err := os.Stat(fullPath)
		fileExists := err == nil

		// If the stored path is already absolute, also try it as-is
		if !fileExists && filepath.IsAbs(track.path) {
			_, err2 := os.Stat(track.path)
			fileExists = err2 == nil
		}

		if !fileExists {
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

	return stats, nil
}
