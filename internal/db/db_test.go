package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConnection(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	db, err := Connect(context.Background(), dbURL, 5)
	if err != nil {
		t.Skipf("Cannot connect: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

func TestGetTrackStats(t *testing.T) {
	db := getDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	stats, err := db.GetTrackStats(context.Background())
	if err != nil {
		t.Fatalf("GetTrackStats() error = %v", err)
	}

	keys := []string{"total", "pending", "analyzing", "completed", "failed", "missing"}
	for _, key := range keys {
		if _, ok := stats[key]; !ok {
			t.Errorf("missing key %q", key)
		}
	}
}

func TestCreateTrack(t *testing.T) {
	db := getDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx := context.Background()
	id, err := db.CreateTrack(ctx, "/test/song.mp3", "song.mp3")
	if err != nil {
		t.Fatalf("CreateTrack() error = %v", err)
	}
	defer db.Pool().Exec(ctx, "DELETE FROM tracks WHERE id = $1", id)

	if id <= 0 {
		t.Errorf("CreateTrack() id = %d, want > 0", id)
	}
}

func TestUpdateTrackStatus(t *testing.T) {
	db := getDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx := context.Background()
	id, _ := db.CreateTrack(ctx, "/test/update.mp3", "update.mp3")
	defer db.Pool().Exec(ctx, "DELETE FROM tracks WHERE id = $1", id)

	err := db.UpdateTrackStatus(ctx, id, "analyzing", nil)
	if err != nil {
		t.Errorf("UpdateTrackStatus() error = %v", err)
	}

	err = db.UpdateTrackStatus(ctx, id, "completed", nil)
	if err != nil {
		t.Errorf("UpdateTrackStatus() error = %v", err)
	}
}

func TestScheduleTrackRetryAndExhaustion(t *testing.T) {
	database := getDB(t)
	if database == nil {
		return
	}
	defer database.Close()
	ctx := context.Background()
	id, err := database.CreateTrack(ctx, filepath.Join(t.TempDir(), "retry.mp3"), "retry.mp3")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Pool().Exec(ctx, "DELETE FROM tracks WHERE id = $1", id)

	if _, err := database.Pool().Exec(ctx, `UPDATE tracks SET analysis_status='analyzing', analysis_attempts=1, analysis_started_at=NOW() WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	status, err := database.ScheduleTrackRetry(ctx, id, 2, time.Hour, "temporary analyzer failure")
	if err != nil || status != "pending" {
		t.Fatalf("first retry status=%q err=%v", status, err)
	}
	var attempts int
	if err := database.Pool().QueryRow(ctx, "SELECT analysis_attempts FROM tracks WHERE id=$1", id).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("attempts=%d, want 1", attempts)
	}
	if _, err := database.Pool().Exec(ctx, "UPDATE tracks SET next_retry_at=NOW() WHERE id=$1", id); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Pool().Exec(ctx, `UPDATE tracks SET analysis_status='analyzing', analysis_attempts=2, analysis_started_at=NOW(), next_retry_at=NULL WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	status, err = database.ScheduleTrackRetry(ctx, id, 2, time.Hour, "still failing")
	if err != nil || status != "failed" {
		t.Fatalf("exhausted retry status=%q err=%v", status, err)
	}
}

func TestRecoverStaleAnalysisClaim(t *testing.T) {
	database := getDB(t)
	if database == nil {
		return
	}
	defer database.Close()
	ctx := context.Background()
	id, err := database.CreateTrack(ctx, filepath.Join(t.TempDir(), "stale.mp3"), "stale.mp3")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Pool().Exec(ctx, "DELETE FROM tracks WHERE id = $1", id)
	if _, err := database.Pool().Exec(ctx, `UPDATE tracks SET analysis_status='analyzing', analysis_attempts=1, analysis_started_at=NOW()-INTERVAL '1 hour' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	recovered, err := database.RecoverStaleAnalysis(ctx, time.Minute, 5, time.Second)
	if err != nil || recovered < 1 {
		t.Fatalf("recovered=%d err=%v", recovered, err)
	}
	var status string
	if err := database.Pool().QueryRow(ctx, "SELECT analysis_status FROM tracks WHERE id=$1", id).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "pending" {
		t.Fatalf("status=%q, want pending", status)
	}
}

func TestSaveAudioFeatures(t *testing.T) {
	db := getDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx := context.Background()
	id, _ := db.CreateTrack(ctx, "/test/features.mp3", "features.mp3")
	defer db.Pool().Exec(ctx, "DELETE FROM audio_features WHERE track_id = $1", id)
	defer db.Pool().Exec(ctx, "DELETE FROM tracks WHERE id = $1", id)

	bpm := 120.0
	key := "C"
	features := &AudioFeatures{
		TrackID: id,
		BPM:     &bpm,
		Key:     &key,
	}

	err := db.SaveAudioFeatures(ctx, features)
	if err != nil {
		t.Errorf("SaveAudioFeatures() error = %v", err)
	}
}

func getDB(t *testing.T) *DB {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
		return nil
	}

	db, err := Connect(context.Background(), dbURL, 5)
	if err != nil {
		t.Skipf("Cannot connect: %v", err)
		return nil
	}
	return db
}
