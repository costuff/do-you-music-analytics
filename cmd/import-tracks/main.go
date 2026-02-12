package main

import (
"context"
"fmt"
"os"
"path/filepath"
"strings"
"sync"
"sync/atomic"

	version := getVersion()
	log.Info(fmt.Sprintf("Track Importer v%s", version))
	
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed: %v", err))
	}
	
	musicDir := cfg.NASMountPath
	if len(os.Args) > 1 {
		musicDir = os.Args[1]
	}
	
	log.Info(fmt.Sprintf("Music Dir: %s", musicDir))
	
	ctx := context.Background()
	database, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		log.Fatal(fmt.Sprintf("DB Failed: %v", err))
	}
	defer database.Close()
	
	log.Info("Connected to DB")
	log.Info("Scanning...")
	
	mp3Files, err := findMP3Files(musicDir)
	if err != nil {
		log.Fatal(fmt.Sprintf("Scan failed: %v", err))
	}
	
	log.Info(fmt.Sprintf("Found %d MP3 files", len(mp3Files)))
	
	if len(mp3Files) == 0 {
		return
	}
	
	log.Info("Importing...")
	imported, skipped, failed := importTracks(ctx, database, musicDir, mp3Files, log)
	log.Info(fmt.Sprintf("Done: %d imported, %d skipped, %d failed", imported, skipped, failed))
}

func findMP3Files(rootDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
if err != nil {
return nil
}
if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mp3") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func importTracks(ctx context.Context, database *db.DB, baseDir string, files []string, log *logger.Logger) (int64, int64, int64) {
	var importedCount, skippedCount, failedCount atomic.Int64
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)
	
	for _, filePath := range files {
		wg.Add(1)
		sem <- struct{}{}
		go func(path string) {
			defer wg.Done()
			defer func() { <-sem }()
			
			relPath, _ := filepath.Rel(baseDir, path)
			fileName := filepath.Base(path)
			
			existing, _ := database.GetTrackByPath(ctx, relPath)
			if existing != nil {
				skippedCount.Add(1)
				return
			}
			
			_, err := database.CreateTrack(ctx, relPath, fileName)
			if err != nil {
				if strings.Contains(err.Error(), "duplicate") {
					skippedCount.Add(1)
					return
				}
				failedCount.Add(1)
				return
			}
			
			importedCount.Add(1)
		}(filePath)
	}
	
	wg.Wait()
	return importedCount.Load(), skippedCount.Load(), failedCount.Load()
}

func getVersion() string {
	version, _ := os.ReadFile("VERSION")
	return strings.TrimSpace(string(version))
}
