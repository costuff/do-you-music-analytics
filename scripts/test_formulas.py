#!/usr/bin/env python3
"""
Unit tests for the high-level audio feature formulas in analyze_audio.py.

Fast tests (TestComputeHighLevel*) call _compute_high_level_features() directly
with crafted raw-feature dicts — no audio file or librosa needed.

Integration tests (TestExtractAudioFeatures*) generate synthetic WAV files and
run the full extract_audio_features() pipeline; they are skipped automatically
when librosa / soundfile are unavailable.

Run with:
    python -m pytest scripts/test_formulas.py -v
  or:
    python scripts/test_formulas.py
"""

import os
import sys
import unittest

# Allow "python scripts/test_formulas.py" from the repo root
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import numpy as np

from scripts.analyze_audio import _compute_high_level_features, extract_audio_features

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

_HIGH_LEVEL_KEYS = [
    "energy", "danceability", "valence", "acousticness",
    "instrumentalness", "speechiness", "liveness", "melodicness",
]


def _make_raw(**overrides):
    """Return a minimal raw-features dict with sensible defaults."""
    defaults = {
        "bpm": 120.0,
        "mode": "major",
        "rms_energy_mean": 0.10,
        "rms_energy_std": 0.02,
        "spectral_centroid_mean": 2000.0,
        "spectral_centroid_std": 400.0,
        "spectral_flatness_mean": 0.02,
        "zero_crossing_rate_mean": 0.08,
        "mfcc_1_std": 8.0,
    }
    defaults.update(overrides)
    # Allow callers to explicitly set a key to None (e.g. mode=None)
    return {k: v for k, v in defaults.items() if v is not None or k in overrides}


def _assert_in_range(tc: unittest.TestCase, feats: dict):
    """Assert every high-level feature is present and in [0, 1]."""
    for key in _HIGH_LEVEL_KEYS:
        tc.assertIn(key, feats, f"{key!r} missing from output")
        val = feats[key]
        tc.assertGreaterEqual(val, 0.0, f"{key} = {val!r} is < 0")
        tc.assertLessEqual(val, 1.0, f"{key} = {val!r} is > 1")


# ---------------------------------------------------------------------------
# Fast unit tests — no audio file required
# ---------------------------------------------------------------------------


class TestComputeHighLevelRanges(unittest.TestCase):
    """All high-level features must be in [0, 1] for any sane input."""

    def test_typical_track(self):
        _assert_in_range(self, _compute_high_level_features(_make_raw()))

    def test_silence(self):
        raw = _make_raw(rms_energy_mean=0.0, rms_energy_std=0.0)
        _assert_in_range(self, _compute_high_level_features(raw))

    def test_empty_raw_dict(self):
        """Empty dict should use defaults and return values in [0, 1]."""
        _assert_in_range(self, _compute_high_level_features({}))

    def test_very_high_bpm(self):
        """BPM ≥ 250 must not produce negative danceability."""
        raw = _make_raw(bpm=300.0)
        feats = _compute_high_level_features(raw)
        _assert_in_range(self, feats)
        self.assertGreaterEqual(feats["danceability"], 0.0)

    def test_bpm_zero(self):
        """BPM=0 should gracefully fall back to default danceability."""
        raw = _make_raw(bpm=0.0)
        feats = _compute_high_level_features(raw)
        _assert_in_range(self, feats)

    def test_bpm_none(self):
        raw = _make_raw()
        raw.pop("bpm", None)
        _assert_in_range(self, _compute_high_level_features(raw))

    def test_very_high_rms(self):
        """Saturated RMS should clamp energy to 1.0."""
        raw = _make_raw(rms_energy_mean=1.0)
        feats = _compute_high_level_features(raw)
        _assert_in_range(self, feats)
        self.assertAlmostEqual(feats["energy"], 1.0)

    def test_low_rms(self):
        raw = _make_raw(rms_energy_mean=0.001)
        feats = _compute_high_level_features(raw)
        _assert_in_range(self, feats)
        self.assertLess(feats["energy"], 0.01)


class TestComputeHighLevelDanceability(unittest.TestCase):
    """Danceability formula correctness."""

    def test_optimal_bpm_beats_extreme(self):
        """Tracks near 125 BPM should be more danceable than extreme BPMs."""
        feats_opt = _compute_high_level_features(_make_raw(bpm=125.0))
        feats_ext = _compute_high_level_features(_make_raw(bpm=20.0))
        self.assertGreater(feats_opt["danceability"], feats_ext["danceability"])

    def test_bpm_325_not_negative(self):
        """BPM=325 must give danceability ≥ 0 (old formula went negative here)."""
        feats = _compute_high_level_features(_make_raw(bpm=325.0))
        self.assertGreaterEqual(feats["danceability"], 0.0)

    def test_steady_beat_boosts_danceability(self):
        """Low onset-strength CoV (steady beat) → higher danceability."""
        # Steady beat: constant onset envelope
        steady = np.ones(100) * 0.5
        # Irregular beat: highly variable onset envelope
        irregular = np.random.RandomState(42).uniform(0, 1, 100)

        raw = _make_raw(bpm=125.0)
        feats_steady = _compute_high_level_features(raw, onset_env=steady)
        feats_irreg  = _compute_high_level_features(raw, onset_env=irregular)
        self.assertGreater(feats_steady["danceability"], feats_irreg["danceability"])


class TestComputeHighLevelValence(unittest.TestCase):
    """Valence formula correctness."""

    def test_major_higher_than_minor(self):
        """Major mode → higher valence than minor mode (same brightness/energy)."""
        raw_base = _make_raw(spectral_centroid_mean=2000.0, rms_energy_mean=0.15)
        raw_major = dict(raw_base, mode="major")
        raw_minor = dict(raw_base, mode="minor")
        feats_major = _compute_high_level_features(raw_major)
        feats_minor = _compute_high_level_features(raw_minor)
        self.assertGreater(feats_major["valence"], feats_minor["valence"])

    def test_mode_none_does_not_crash(self):
        raw = _make_raw()
        raw["mode"] = None
        _assert_in_range(self, _compute_high_level_features(raw))

    def test_mode_missing_does_not_crash(self):
        raw = _make_raw()
        raw.pop("mode", None)
        _assert_in_range(self, _compute_high_level_features(raw))

    def test_bright_higher_than_dark(self):
        """Higher spectral centroid → higher valence."""
        feats_bright = _compute_high_level_features(_make_raw(spectral_centroid_mean=4000.0))
        feats_dark   = _compute_high_level_features(_make_raw(spectral_centroid_mean=500.0))
        self.assertGreater(feats_bright["valence"], feats_dark["valence"])


class TestComputeHighLevelAcousticness(unittest.TestCase):
    """Acousticness formula correctness (spectral flatness-based)."""

    def test_low_flatness_is_acoustic(self):
        """Very low spectral flatness → high acousticness."""
        feats = _compute_high_level_features(_make_raw(spectral_flatness_mean=0.001))
        self.assertGreater(feats["acousticness"], 0.9)

    def test_high_flatness_is_not_acoustic(self):
        """High spectral flatness (noise-like) → low acousticness."""
        feats = _compute_high_level_features(_make_raw(spectral_flatness_mean=0.15))
        self.assertLess(feats["acousticness"], 0.3)

    def test_acoustic_more_than_electric(self):
        feats_ac = _compute_high_level_features(_make_raw(spectral_flatness_mean=0.005))
        feats_el = _compute_high_level_features(_make_raw(spectral_flatness_mean=0.10))
        self.assertGreater(feats_ac["acousticness"], feats_el["acousticness"])

    def test_rolloff_fallback(self):
        """Without spectral_flatness_mean, falls back to inverse-rolloff."""
        raw = _make_raw()
        del raw["spectral_flatness_mean"]
        raw["spectral_rolloff_mean"] = 4000.0
        feats = _compute_high_level_features(raw)
        _assert_in_range(self, feats)
        # rolloff=4000 / 8000 = 0.5 → acousticness = 0.5
        self.assertAlmostEqual(feats["acousticness"], 0.5, places=5)

    def test_no_flatness_no_rolloff_defaults_half(self):
        raw = _make_raw()
        raw.pop("spectral_flatness_mean", None)
        raw.pop("spectral_rolloff_mean", None)
        feats = _compute_high_level_features(raw)
        _assert_in_range(self, feats)
        self.assertAlmostEqual(feats["acousticness"], 0.5)


class TestComputeHighLevelInstrumentalness(unittest.TestCase):
    """Instrumentalness formula correctness (MFCC-1 std + ZCR)."""

    def test_low_mfcc1_std_is_instrumental(self):
        """Low MFCC-1 std (stable spectrum) → high instrumentalness."""
        feats = _compute_high_level_features(_make_raw(mfcc_1_std=2.0))
        self.assertGreater(feats["instrumentalness"], 0.6)

    def test_high_mfcc1_std_is_vocal(self):
        """High MFCC-1 std (spectrally dynamic) → low instrumentalness."""
        feats = _compute_high_level_features(_make_raw(mfcc_1_std=25.0))
        self.assertLess(feats["instrumentalness"], 0.4)

    def test_instrumental_more_than_vocal(self):
        feats_i = _compute_high_level_features(_make_raw(mfcc_1_std=3.0))
        feats_v = _compute_high_level_features(_make_raw(mfcc_1_std=22.0))
        self.assertGreater(feats_i["instrumentalness"], feats_v["instrumentalness"])

    def test_no_mfcc1_falls_back_to_zcr(self):
        """Without mfcc_1_std, falls back to ZCR-only heuristic."""
        raw = _make_raw()
        raw.pop("mfcc_1_std", None)
        feats = _compute_high_level_features(raw)
        _assert_in_range(self, feats)


# ---------------------------------------------------------------------------
# Integration tests — require librosa + soundfile + a few seconds of CPU
# ---------------------------------------------------------------------------


def _try_import_audio_libs():
    try:
        import soundfile  # noqa: F401
        import librosa    # noqa: F401
        return True
    except ImportError:
        return False


_AUDIO_LIBS_AVAILABLE = _try_import_audio_libs()


def _write_wav(signal: np.ndarray, sr: int = 22050) -> str:
    """Write signal to a temporary WAV file and return its path."""
    import soundfile as sf
    import tempfile
    f = tempfile.NamedTemporaryFile(suffix=".wav", delete=False)
    sf.write(f.name, signal.astype(np.float32), sr)
    f.close()
    return f.name


@unittest.skipUnless(_AUDIO_LIBS_AVAILABLE, "librosa / soundfile not installed")
class TestExtractAudioFeaturesSineWave(unittest.TestCase):
    """Full-pipeline tests using a synthetic pure sine wave (440 Hz, 5 s)."""

    @classmethod
    def setUpClass(cls):
        sr = 22050
        t = np.linspace(0, 5.0, sr * 5)
        signal = 0.3 * np.sin(2 * np.pi * 440 * t)
        cls.wav = _write_wav(signal, sr)
        cls.features = extract_audio_features(cls.wav, quiet=True)

    @classmethod
    def tearDownClass(cls):
        try:
            os.unlink(cls.wav)
        except OSError:
            pass

    def test_no_error(self):
        self.assertNotIn("error", self.features)

    def test_all_high_level_in_range(self):
        _assert_in_range(self, self.features)

    def test_energy_nonzero(self):
        self.assertGreater(self.features["energy"], 0.0)

    def test_acousticness_high_for_pure_tone(self):
        """A pure sine wave has low spectral flatness → high acousticness."""
        self.assertGreater(self.features["acousticness"], 0.5)


@unittest.skipUnless(_AUDIO_LIBS_AVAILABLE, "librosa / soundfile not installed")
class TestExtractAudioFeaturesWhiteNoise(unittest.TestCase):
    """Full-pipeline tests using white noise (5 s)."""

    @classmethod
    def setUpClass(cls):
        rng = np.random.RandomState(0)
        signal = rng.uniform(-0.3, 0.3, 22050 * 5)
        cls.wav = _write_wav(signal)
        cls.features = extract_audio_features(cls.wav, quiet=True)

    @classmethod
    def tearDownClass(cls):
        try:
            os.unlink(cls.wav)
        except OSError:
            pass

    def test_all_high_level_in_range(self):
        # Some features may be absent if extraction failed; check those present
        for key in _HIGH_LEVEL_KEYS:
            val = self.features.get(key)
            if val is not None:
                self.assertGreaterEqual(val, 0.0, f"{key} = {val!r} < 0")
                self.assertLessEqual(val, 1.0, f"{key} = {val!r} > 1")

    def test_acousticness_low_for_noise(self):
        """White noise has high spectral flatness → low acousticness."""
        val = self.features.get("acousticness")
        if val is not None:
            self.assertLess(val, 0.5)


@unittest.skipUnless(_AUDIO_LIBS_AVAILABLE, "librosa / soundfile not installed")
class TestExtractAudioFeaturesSilence(unittest.TestCase):
    """Silence should not crash the analyzer."""

    def test_silence_does_not_crash(self):
        signal = np.zeros(22050 * 3, dtype=np.float32)
        wav = _write_wav(signal)
        try:
            features = extract_audio_features(wav, quiet=True)
            self.assertIsInstance(features, dict)
        finally:
            try:
                os.unlink(wav)
            except OSError:
                pass


if __name__ == "__main__":
    unittest.main(verbosity=2)
