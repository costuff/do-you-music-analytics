package db

import (
	"math"
	"testing"
)

func TestCosineSim_IdenticalVectors(t *testing.T) {
	v := []float64{0.4, 0.8, 0.2, 0.1, 0.5, 0.3, 0.7, 0.6}
	got := cosineSim(v, v)
	if math.Abs(got-1.0) > 1e-9 {
		t.Errorf("identical vectors: want 1.0, got %v", got)
	}
}

func TestCosineSim_OppositeVectors(t *testing.T) {
	a := []float64{1, 0, 0, 0, 0, 0, 0, 0}
	b := []float64{0, 1, 0, 0, 0, 0, 0, 0}
	got := cosineSim(a, b)
	if math.Abs(got) > 1e-9 {
		t.Errorf("orthogonal vectors: want 0.0, got %v", got)
	}
}

func TestCosineSim_ZeroVector(t *testing.T) {
	a := []float64{0, 0, 0, 0, 0, 0, 0, 0}
	b := []float64{0.5, 0.3, 0.1, 0.8, 0.2, 0.4, 0.6, 0.7}
	got := cosineSim(a, b)
	if got != 0 {
		t.Errorf("zero vector: want 0.0, got %v", got)
	}
}

func TestCosineSim_KnownValue(t *testing.T) {
	// [1,0] · [1,1]/|[1,1]| = 1/sqrt(2) ≈ 0.7071
	a := []float64{1, 0}
	b := []float64{1, 1}
	got := cosineSim(a, b)
	want := 1.0 / math.Sqrt(2)
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("known value: want %v, got %v", want, got)
	}
}

func TestCosineSim_SymmetricProperty(t *testing.T) {
	a := []float64{0.3, 0.7, 0.1, 0.9, 0.4, 0.2, 0.8, 0.5}
	b := []float64{0.6, 0.2, 0.5, 0.1, 0.8, 0.3, 0.4, 0.7}
	if cosineSim(a, b) != cosineSim(b, a) {
		t.Error("cosine similarity must be symmetric")
	}
}

func TestToVector_Length(t *testing.T) {
	e := TrackEmotions{
		Tension: 0.1, Melancholy: 0.2, Euphoria: 0.3, Aggression: 0.4,
		Serenity: 0.5, Nostalgia: 0.6, Sensuality: 0.7, Power: 0.8,
	}
	v := toVector(e)
	if len(v) != 8 {
		t.Errorf("toVector: want len=8, got %d", len(v))
	}
}

func TestToVector_ValuesMatchFields(t *testing.T) {
	e := TrackEmotions{
		Tension: 0.1, Melancholy: 0.2, Euphoria: 0.3, Aggression: 0.4,
		Serenity: 0.5, Nostalgia: 0.6, Sensuality: 0.7, Power: 0.8,
	}
	v := toVector(e)
	expected := []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}
	for i, want := range expected {
		if v[i] != want {
			t.Errorf("toVector[%d]: want %v, got %v", i, want, v[i])
		}
	}
}

// TestComputeSimilarity_Integration is skipped without a real DB.
func TestComputeSimilarity_Integration(t *testing.T) {
	t.Skip("requires DATABASE_URL")
}

func TestGetSimilarTracks_Integration(t *testing.T) {
	t.Skip("requires DATABASE_URL")
}

func TestGetAllEmotionVectors_Integration(t *testing.T) {
	t.Skip("requires DATABASE_URL")
}

func TestGetSimilarityStats_Integration(t *testing.T) {
	t.Skip("requires DATABASE_URL")
}
