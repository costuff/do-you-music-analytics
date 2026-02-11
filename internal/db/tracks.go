package db

import "context"

type Track struct {
ID int
FilePath string
AnalysisStatus string
}

func (db *DB) GetTrackStats(ctx context.Context) (map[string]int, error) {
stats := make(map[string]int)
stats["total"] = 0
stats["pending"] = 0
stats["completed"] = 0
stats["failed"] = 0
stats["analyzing"] = 0
return stats, nil
}

func (db *DB) GetPendingTracks(ctx context.Context, limit int) ([]*Track, error) {
return []*Track{}, nil
}
