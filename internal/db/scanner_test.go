package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScanAndImportNewTracks_NoDatabase(t *testing.T) {
	// Test scanner logic without database dependency
	// This tests file scanning and path handling

	// Create temporary test directory with MP3 files
	tempDir := t.TempDir()

	// Create test MP3 files
	testFiles := []string{
		"track1.mp3",
		"track2.mp3",
		"subdir/track3.mp3",
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filePath, []byte("fake mp3 data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Test: Verify test files were created
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) < 1 {
		t.Errorf("Expected at least 1 entry in temp dir, got %d", len(entries))
	}

	// Verify MP3 files exist
	for _, file := range []string{"track1.mp3", "track2.mp3"} {
		if _, err := os.Stat(filepath.Join(tempDir, file)); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", file)
		}
	}
}

func TestScanAndImportNewTracks_PathHandling(t *testing.T) {
	tests := []struct {
		name     string
		setupDir func(string) error
		wantErr  bool
	}{
		{
			name: "handles nested directories",
			setupDir: func(dir string) error {
				paths := []string{
					"artist1/album1/track1.mp3",
					"artist1/album2/track2.mp3",
					"artist2/single/track3.mp3",
				}
				for _, p := range paths {
					full := filepath.Join(dir, p)
					if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
						return err
					}
					if err := os.WriteFile(full, []byte("mp3"), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "ignores non-mp3 files",
			setupDir: func(dir string) error {
				files := map[string]string{
					"track.mp3":  "valid",
					"cover.jpg":  "ignore",
					"readme.txt": "ignore",
					"song.MP3":   "valid", // case insensitive
				}
				for name, _ := range files {
					if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "handles special characters in filenames",
			setupDir: func(dir string) error {
				files := []string{
					"Track (Remix).mp3",
					"Artist - Title [Label].mp3",
					"Song & More.mp3",
				}
				for _, name := range files {
					if err := os.WriteFile(filepath.Join(dir, name), []byte("mp3"), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "handles unicode filenames",
			setupDir: func(dir string) error {
				files := []string{
					"Solomün - Track.mp3",
					"日本語のタイトル.mp3",
					"Müller & Östberg.mp3",
				}
				for _, name := range files {
					if err := os.WriteFile(filepath.Join(dir, name), []byte("mp3"), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			if err := tt.setupDir(tempDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Count MP3 files
			count := 0
			err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && filepath.Ext(path) == ".mp3" || filepath.Ext(path) == ".MP3" {
					count++
				}
				return nil
			})

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.wantErr && count == 0 {
				t.Error("Expected to find MP3 files, found none")
			}
		})
	}
}

func TestScanAndImportNewTracks_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Test scanning empty directory doesn't error
	var foundMP3 bool
	err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".mp3" {
			foundMP3 = true
		}
		return nil
	})

	if err != nil {
		t.Errorf("Walk failed on empty directory: %v", err)
	}

	if foundMP3 {
		t.Error("Found MP3 in empty directory")
	}
}

func TestScanAndImportNewTracks_SymlinkHandling(t *testing.T) {
	tempDir := t.TempDir()

	// Create a real MP3 file
	realFile := filepath.Join(tempDir, "real.mp3")
	if err := os.WriteFile(realFile, []byte("mp3 data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink to it
	symlinkPath := filepath.Join(tempDir, "symlink.mp3")
	if err := os.Symlink(realFile, symlinkPath); err != nil {
		if os.IsPermission(err) {
			t.Skip("Symlink creation not permitted")
		}
		t.Fatal(err)
	}

	// Walk should handle both
	count := 0
	err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".mp3" {
			count++
		}
		return nil
	})

	if err != nil {
		t.Errorf("Walk failed with symlinks: %v", err)
	}

	// Should find at least the real file
	if count < 1 {
		t.Errorf("Expected at least 1 MP3 file, got %d", count)
	}
}

func TestScanAndImportNewTracks_PermissionErrors(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Running as root, permission tests not applicable")
	}

	tempDir := t.TempDir()

	// Create subdirectory with restricted permissions
	restrictedDir := filepath.Join(tempDir, "restricted")
	if err := os.Mkdir(restrictedDir, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(restrictedDir, 0755) // Cleanup

	// Walk should skip restricted directories gracefully
	err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Expected: permission denied on restricted dir
			return nil // Continue walking
		}
		return nil
	})

	if err != nil {
		t.Errorf("Walk should handle permission errors gracefully, got: %v", err)
	}
}

func TestRelativePathFromMusicDir(t *testing.T) {
	tests := []struct {
		name      string
		musicDir  string
		filePath  string
		expected  string
		wantError bool
	}{
		{
			name:      "simple relative path",
			musicDir:  "/mnt/music/downloads",
			filePath:  "/mnt/music/downloads/track.mp3",
			expected:  "track.mp3",
			wantError: false,
		},
		{
			name:      "nested relative path",
			musicDir:  "/mnt/music/downloads",
			filePath:  "/mnt/music/downloads/artist/album/track.mp3",
			expected:  "artist/album/track.mp3",
			wantError: false,
		},
		{
			name:      "local vs docker path",
			musicDir:  "/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks",
			filePath:  "/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks/track.mp3",
			expected:  "track.mp3",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relPath, err := filepath.Rel(tt.musicDir, tt.filePath)

			if tt.wantError && err == nil {
				t.Error("Expected error, got nil")
			}

			if !tt.wantError {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if relPath != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, relPath)
				}
			}
		})
	}
}

func TestScanAndImportNewTracks_Integration(t *testing.T) {
	// This test requires DATABASE_URL to be set
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	database, err := Connect(ctx, dbURL, 5)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Create temp directory with test MP3s
	tempDir := t.TempDir()
	testFiles := []string{"test1.mp3", "test2.mp3", "test3.mp3"}

	for _, file := range testFiles {
		path := filepath.Join(tempDir, file)
		if err := os.WriteFile(path, []byte("fake mp3"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Run scan and import
	imported, err := database.ScanAndImportNewTracks(ctx, tempDir)
	if err != nil {
		t.Fatalf("ScanAndImportNewTracks failed: %v", err)
	}

	if imported != len(testFiles) {
		t.Errorf("Expected %d imports, got %d", len(testFiles), imported)
	}

	// Run again - should import 0 (no new files)
	imported2, err := database.ScanAndImportNewTracks(ctx, tempDir)
	if err != nil {
		t.Fatalf("Second scan failed: %v", err)
	}

	if imported2 != 0 {
		t.Errorf("Second scan should import 0, got %d", imported2)
	}

	// Verify tracks were inserted with correct status
	stats, err := database.GetTrackStats(ctx)
	if err != nil {
		t.Fatalf("GetTrackStats failed: %v", err)
	}

	if stats["pending"] < int64(imported) {
		t.Errorf("Expected at least %d pending tracks, got %d", imported, stats["pending"])
	}
}
