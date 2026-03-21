package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// TaxonomyNode is one item in the hierarchical label tree.
// Children is populated when building the full tree.
type TaxonomyNode struct {
	ID          int             `json:"id"`
	ParentID    *int            `json:"parent_id,omitempty"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Source      string          `json:"source"`
	Children    []*TaxonomyNode `json:"children,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// TrackLabel stores an auto-computed path and an optional user-curated path.
// label_path is the user curation; auto_label_path is the system suggestion.
type TrackLabel struct {
	ID            int64     `json:"id"`
	TrackID       int64     `json:"track_id"`
	LabelPath     []string  `json:"label_path"`
	AutoLabelPath []string  `json:"auto_label_path"`
	CuratedBy     string    `json:"curated_by,omitempty"`
	Notes         string    `json:"notes,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// GetTaxonomyTree returns the full taxonomy as a nested tree (roots → children).
func (db *DB) GetTaxonomyTree(ctx context.Context) ([]*TaxonomyNode, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, parent_id, name, COALESCE(description,''), source, created_at
		FROM taxonomy_nodes ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("query taxonomy: %w", err)
	}
	defer rows.Close()

	byID := map[int]*TaxonomyNode{}
	var roots []*TaxonomyNode

	for rows.Next() {
		n := &TaxonomyNode{}
		if err := rows.Scan(&n.ID, &n.ParentID, &n.Name, &n.Description, &n.Source, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan taxonomy node: %w", err)
		}
		byID[n.ID] = n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, n := range byID {
		if n.ParentID == nil {
			roots = append(roots, n)
		} else if parent, ok := byID[*n.ParentID]; ok {
			parent.Children = append(parent.Children, n)
		}
	}
	return roots, nil
}

// AddTaxonomyNode creates a new node under an optional parent.
func (db *DB) AddTaxonomyNode(ctx context.Context, parentID *int, name, description, source string) (*TaxonomyNode, error) {
	n := &TaxonomyNode{}
	err := db.pool.QueryRow(ctx, `
		INSERT INTO taxonomy_nodes (parent_id, name, description, source)
		VALUES ($1, $2, $3, $4)
		RETURNING id, parent_id, name, COALESCE(description,''), source, created_at
	`, parentID, name, description, source).Scan(
		&n.ID, &n.ParentID, &n.Name, &n.Description, &n.Source, &n.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert taxonomy node: %w", err)
	}
	return n, nil
}

// GetTrackLabel returns the stored label for a track, or nil if none exists yet.
func (db *DB) GetTrackLabel(ctx context.Context, trackID int64) (*TrackLabel, error) {
	label := &TrackLabel{}
	var lpJSON, apJSON []byte
	err := db.pool.QueryRow(ctx, `
		SELECT id, track_id, label_path, auto_label_path,
		       COALESCE(curated_by,''), COALESCE(notes,''), created_at
		FROM track_labels WHERE track_id = $1
	`, trackID).Scan(
		&label.ID, &label.TrackID, &lpJSON, &apJSON,
		&label.CuratedBy, &label.Notes, &label.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get track label: %w", err)
	}
	label.LabelPath = unmarshalStringSlice(lpJSON)
	label.AutoLabelPath = unmarshalStringSlice(apJSON)
	return label, nil
}

// SetTrackLabel upserts a user-curated label for a track.
// The auto_label_path is preserved from the existing row if already computed.
func (db *DB) SetTrackLabel(ctx context.Context, trackID int64, labelPath []string, curatedBy, notes string) (*TrackLabel, error) {
	lpJSON, _ := json.Marshal(labelPath)

	// Keep previously computed auto path if it exists; otherwise derive it now.
	autoPath := db.resolveAutoLabel(ctx, trackID)
	apJSON, _ := json.Marshal(autoPath)

	label := &TrackLabel{}
	var lpOut, apOut []byte
	err := db.pool.QueryRow(ctx, `
		INSERT INTO track_labels (track_id, label_path, auto_label_path, curated_by, notes)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (track_id) DO UPDATE SET
			label_path = EXCLUDED.label_path,
			curated_by = EXCLUDED.curated_by,
			notes      = EXCLUDED.notes
		RETURNING id, track_id, label_path, auto_label_path,
		          COALESCE(curated_by,''), COALESCE(notes,''), created_at
	`, trackID, lpJSON, apJSON, curatedBy, notes).Scan(
		&label.ID, &label.TrackID, &lpOut, &apOut,
		&label.CuratedBy, &label.Notes, &label.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("set track label: %w", err)
	}
	label.LabelPath = unmarshalStringSlice(lpOut)
	label.AutoLabelPath = unmarshalStringSlice(apOut)
	return label, nil
}

// AutoLabelTrack upserts the auto_label_path for a track using its dominant zone.
// The user label_path is not overwritten.
func (db *DB) AutoLabelTrack(ctx context.Context, trackID int64) (*TrackLabel, error) {
	path := db.resolveAutoLabel(ctx, trackID)
	pathJSON, _ := json.Marshal(path)

	label := &TrackLabel{}
	var lpOut, apOut []byte
	err := db.pool.QueryRow(ctx, `
		INSERT INTO track_labels (track_id, auto_label_path, label_path)
		VALUES ($1, $2, $2)
		ON CONFLICT (track_id) DO UPDATE SET auto_label_path = EXCLUDED.auto_label_path
		RETURNING id, track_id, label_path, auto_label_path,
		          COALESCE(curated_by,''), COALESCE(notes,''), created_at
	`, trackID, pathJSON).Scan(
		&label.ID, &label.TrackID, &lpOut, &apOut,
		&label.CuratedBy, &label.Notes, &label.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("auto label track %d: %w", trackID, err)
	}
	label.LabelPath = unmarshalStringSlice(lpOut)
	label.AutoLabelPath = unmarshalStringSlice(apOut)
	return label, nil
}

// GetLabelsByPath returns up to 200 track labels whose label_path contains
// all elements of path (JSONB containment: label_path @> path).
func (db *DB) GetLabelsByPath(ctx context.Context, path []string) ([]TrackLabel, error) {
	pathJSON, _ := json.Marshal(path)
	rows, err := db.pool.Query(ctx, `
		SELECT id, track_id, label_path, auto_label_path,
		       COALESCE(curated_by,''), COALESCE(notes,''), created_at
		FROM track_labels
		WHERE label_path @> $1::jsonb
		ORDER BY created_at DESC LIMIT 200
	`, pathJSON)
	if err != nil {
		return nil, fmt.Errorf("query labels by path: %w", err)
	}
	defer rows.Close()

	var out []TrackLabel
	for rows.Next() {
		var lbl TrackLabel
		var lp, ap []byte
		if err := rows.Scan(&lbl.ID, &lbl.TrackID, &lp, &ap, &lbl.CuratedBy, &lbl.Notes, &lbl.CreatedAt); err != nil {
			return nil, err
		}
		lbl.LabelPath = unmarshalStringSlice(lp)
		lbl.AutoLabelPath = unmarshalStringSlice(ap)
		out = append(out, lbl)
	}
	return out, rows.Err()
}

// resolveAutoLabel finds the dominant emotion zone for a track by querying
// track_projections. Falls back to an empty slice if the projection is missing.
func (db *DB) resolveAutoLabel(ctx context.Context, trackID int64) []string {
	var zone string
	_ = db.pool.QueryRow(ctx, //nolint:errcheck // best-effort: returns empty if no projection
		`SELECT zone FROM track_projections WHERE track_id = $1`, trackID,
	).Scan(&zone)
	if zone == "" {
		return []string{}
	}
	return []string{zone}
}

func unmarshalStringSlice(b []byte) []string {
	var s []string
	if err := json.Unmarshal(b, &s); err != nil {
		return []string{}
	}
	return s
}
