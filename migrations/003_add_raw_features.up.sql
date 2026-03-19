-- Persist raw audio features that were computed by analyze_audio.py but previously discarded.
-- Adds: spectral centroid/rolloff, zero crossing rate, RMS energy stats, and MFCCs 0-12 (mean+std).
-- Also enables mode (major/minor) which already had a column in audio_features since migration 001
-- but was never written to — analyze_audio.py now computes it using Krumhansl-Kessler profiles.

ALTER TABLE audio_features
    ADD COLUMN IF NOT EXISTS spectral_centroid_mean  DECIMAL(10, 2),
    ADD COLUMN IF NOT EXISTS spectral_centroid_std   DECIMAL(10, 2),
    ADD COLUMN IF NOT EXISTS spectral_rolloff_mean   DECIMAL(10, 2),
    ADD COLUMN IF NOT EXISTS zero_crossing_rate_mean DECIMAL(8, 6),
    ADD COLUMN IF NOT EXISTS rms_energy_mean         DECIMAL(8, 6),
    ADD COLUMN IF NOT EXISTS rms_energy_std          DECIMAL(8, 6),
    ADD COLUMN IF NOT EXISTS mfcc_0_mean   DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_0_std    DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_1_mean   DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_1_std    DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_2_mean   DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_2_std    DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_3_mean   DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_3_std    DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_4_mean   DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_4_std    DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_5_mean   DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_5_std    DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_6_mean   DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_6_std    DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_7_mean   DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_7_std    DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_8_mean   DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_8_std    DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_9_mean   DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_9_std    DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_10_mean  DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_10_std   DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_11_mean  DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_11_std   DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_12_mean  DECIMAL(8, 4),
    ADD COLUMN IF NOT EXISTS mfcc_12_std   DECIMAL(8, 4);
