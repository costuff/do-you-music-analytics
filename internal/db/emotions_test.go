package db

import (
	"math"
	"testing"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func fp(v float64) *float64 { return &v }
func sp(v string) *string   { return &v }

// defaultFeatures returns a complete AudioFeatures set with all fields populated.
func defaultFeatures() *AudioFeatures {
	return &AudioFeatures{
		TrackID:          1,
		BPM:              fp(120),
		Mode:             sp("major"),
		Energy:           fp(0.6),
		Valence:          fp(0.6),
		Danceability:     fp(0.7),
		Acousticness:     fp(0.3),
		Speechiness:      fp(0.1),
		Liveness:         fp(0.2),
		Melodicness:      fp(0.5),
		Instrumentalness: fp(0.8),
	}
}

// allInRange asserts every emotion field is in [0, 1].
func allInRange(t *testing.T, e TrackEmotions) {
	t.Helper()
	dims := map[string]float64{
		"tension":    e.Tension,
		"melancholy": e.Melancholy,
		"euphoria":   e.Euphoria,
		"aggression": e.Aggression,
		"serenity":   e.Serenity,
		"nostalgia":  e.Nostalgia,
		"sensuality": e.Sensuality,
		"power":      e.Power,
	}
	for name, val := range dims {
		if val < 0 || val > 1 {
			t.Errorf("expected %s in [0,1], got %f", name, val)
		}
		if math.IsNaN(val) || math.IsInf(val, 0) {
			t.Errorf("%s is NaN/Inf", name)
		}
	}
}

// ── range tests ───────────────────────────────────────────────────────────────

func TestComputeEmotions_AllInRange_DefaultFeatures(t *testing.T) {
	e := ComputeEmotions(defaultFeatures())
	allInRange(t, e)
}

func TestComputeEmotions_AllInRange_NilPointers(t *testing.T) {
	// No feature pointers set — should use defaults and not panic.
	e := ComputeEmotions(&AudioFeatures{TrackID: 1})
	allInRange(t, e)
}

func TestComputeEmotions_AllInRange_ExtremeHighEnergy(t *testing.T) {
	f := defaultFeatures()
	f.Energy = fp(1.0)
	f.BPM = fp(200)
	allInRange(t, ComputeEmotions(f))
}

func TestComputeEmotions_AllInRange_ExtremeHighBPM(t *testing.T) {
	f := defaultFeatures()
	f.BPM = fp(300) // bpmNorm clamps at 1.0
	allInRange(t, ComputeEmotions(f))
}

func TestComputeEmotions_AllInRange_ZeroEnergy(t *testing.T) {
	f := defaultFeatures()
	f.Energy = fp(0.0)
	allInRange(t, ComputeEmotions(f))
}

func TestComputeEmotions_AllInRange_MinorMode(t *testing.T) {
	f := defaultFeatures()
	f.Mode = sp("minor")
	allInRange(t, ComputeEmotions(f))
}

func TestComputeEmotions_AllInRange_NilMode(t *testing.T) {
	f := defaultFeatures()
	f.Mode = nil
	allInRange(t, ComputeEmotions(f))
}

// ── semantic correctness tests ────────────────────────────────────────────────

func TestComputeEmotions_HighEnergy_HighTension(t *testing.T) {
	high := defaultFeatures()
	high.Energy = fp(1.0)
	high.Valence = fp(0.1) // low valence boosts tension

	low := defaultFeatures()
	low.Energy = fp(0.1)
	low.Valence = fp(0.9)

	eHigh := ComputeEmotions(high)
	eLow := ComputeEmotions(low)
	if eHigh.Tension <= eLow.Tension {
		t.Errorf("expected high-energy track to have higher tension: %f vs %f",
			eHigh.Tension, eLow.Tension)
	}
}

func TestComputeEmotions_LowValence_HighMelancholy(t *testing.T) {
	sad := defaultFeatures()
	sad.Valence = fp(0.1)
	sad.Energy = fp(0.2)
	sad.Mode = sp("minor")

	happy := defaultFeatures()
	happy.Valence = fp(0.9)
	happy.Energy = fp(0.8)
	happy.Mode = sp("major")

	eSad := ComputeEmotions(sad)
	eHappy := ComputeEmotions(happy)
	if eSad.Melancholy <= eHappy.Melancholy {
		t.Errorf("expected low-valence/minor track to have higher melancholy: %f vs %f",
			eSad.Melancholy, eHappy.Melancholy)
	}
}

func TestComputeEmotions_HighValence_HighEuphoria(t *testing.T) {
	elated := defaultFeatures()
	elated.Valence = fp(1.0)
	elated.Energy = fp(0.9)
	elated.Danceability = fp(0.9)

	gloomy := defaultFeatures()
	gloomy.Valence = fp(0.1)
	gloomy.Energy = fp(0.1)
	gloomy.Danceability = fp(0.1)

	eElated := ComputeEmotions(elated)
	eGloomy := ComputeEmotions(gloomy)
	if eElated.Euphoria <= eGloomy.Euphoria {
		t.Errorf("high-valence/energy track should be more euphoric: %f vs %f",
			eElated.Euphoria, eGloomy.Euphoria)
	}
}

func TestComputeEmotions_HighAcousticness_HighSerenity(t *testing.T) {
	calm := defaultFeatures()
	calm.Acousticness = fp(0.95)
	calm.Energy = fp(0.1)
	calm.Valence = fp(0.7)

	harsh := defaultFeatures()
	harsh.Acousticness = fp(0.05)
	harsh.Energy = fp(0.9)
	harsh.Valence = fp(0.3)

	eCalm := ComputeEmotions(calm)
	eHarsh := ComputeEmotions(harsh)
	if eCalm.Serenity <= eHarsh.Serenity {
		t.Errorf("calm/acoustic track should be more serene: %f vs %f",
			eCalm.Serenity, eHarsh.Serenity)
	}
}

func TestComputeEmotions_HighEnergy_HighPower(t *testing.T) {
	powerful := defaultFeatures()
	powerful.Energy = fp(1.0)
	powerful.Acousticness = fp(0.0)

	quiet := defaultFeatures()
	quiet.Energy = fp(0.0)
	quiet.Acousticness = fp(1.0)

	ePowerful := ComputeEmotions(powerful)
	eQuiet := ComputeEmotions(quiet)
	if ePowerful.Power <= eQuiet.Power {
		t.Errorf("high-energy/low-acoustic track should have more power: %f vs %f",
			ePowerful.Power, eQuiet.Power)
	}
}

func TestComputeEmotions_MajorMode_LowerMelancholy(t *testing.T) {
	major := defaultFeatures()
	major.Mode = sp("major")

	minor := defaultFeatures()
	minor.Mode = sp("minor")

	eMajor := ComputeEmotions(major)
	eMinor := ComputeEmotions(minor)
	// melancholy = 0.5*(1-valence) + 0.3*(1-energy) + 0.2*(1-modeIsMajor)
	// major→ 0.2*0 = 0, minor→ 0.2*1 = 0.2 higher melancholy
	if eMajor.Melancholy >= eMinor.Melancholy {
		t.Errorf("major mode should have lower melancholy: %f vs %f",
			eMajor.Melancholy, eMinor.Melancholy)
	}
}

func TestComputeEmotions_TrackIDPreserved(t *testing.T) {
	f := defaultFeatures()
	f.TrackID = 42
	e := ComputeEmotions(f)
	if e.TrackID != 42 {
		t.Errorf("expected TrackID=42, got %d", e.TrackID)
	}
}

func TestComputeEmotions_ComputedAtNonZero(t *testing.T) {
	e := ComputeEmotions(defaultFeatures())
	if e.ComputedAt.IsZero() {
		t.Error("ComputedAt should be non-zero")
	}
}

// ── formula spot-checks ───────────────────────────────────────────────────────

func TestComputeEmotions_TensionFormula(t *testing.T) {
	// tension = 0.4*energy + 0.3*(1-valence) + 0.2*bpmNorm + 0.1*speechiness
	// With energy=0.5, valence=0.5, bpm=100(→0.5), speechiness=0.0:
	// = 0.4*0.5 + 0.3*0.5 + 0.2*0.5 + 0.1*0.0 = 0.20+0.15+0.10+0.00 = 0.45
	f := &AudioFeatures{
		TrackID:     1,
		Energy:      fp(0.5),
		Valence:     fp(0.5),
		BPM:         fp(100),
		Speechiness: fp(0.0),
		// leave rest nil to use defaults
	}
	e := ComputeEmotions(f)
	want := 0.45
	if math.Abs(e.Tension-want) > 1e-9 {
		t.Errorf("Tension = %f, want %f", e.Tension, want)
	}
}

func TestComputeEmotions_PureMinorFormula_Melancholy(t *testing.T) {
	// melancholy = 0.5*(1-valence) + 0.3*(1-energy) + 0.2*(1-modeIsMajor)
	// With valence=0, energy=0, mode=minor(→0):
	// = 0.5*1 + 0.3*1 + 0.2*1 = 1.0
	f := &AudioFeatures{
		TrackID: 1,
		Valence: fp(0.0),
		Energy:  fp(0.0),
		Mode:    sp("minor"),
	}
	e := ComputeEmotions(f)
	if math.Abs(e.Melancholy-1.0) > 1e-9 {
		t.Errorf("Melancholy = %f, want 1.0", e.Melancholy)
	}
}

func TestComputeEmotions_NilBPM_UsesDefault(t *testing.T) {
	f := defaultFeatures()
	f.BPM = nil // should use bpmNorm=0.5
	e := ComputeEmotions(f)
	allInRange(t, e)
}
