#!/usr/bin/env python3
"""
Music Audio Analysis Script
Extracts BPM, key, audio features, and generates spectrograms from MP3 files.
"""

import os
import sys
import json
import warnings

# Suppress all warnings (including FutureWarnings from librosa)
warnings.filterwarnings('ignore')

import librosa
import numpy as np
import soundfile as sf
from pathlib import Path
import argparse
import matplotlib.pyplot as plt
from typing import Dict, Any, Optional
import signal
from functools import wraps

# Timeout decorator
def timeout(seconds=30):
    def decorator(func):
        def _handle_timeout(signum, frame):
            raise TimeoutError(f"Function call timed out after {seconds} seconds")
        
        @wraps(func)
        def wrapper(*args, **kwargs):
            signal.signal(signal.SIGALRM, _handle_timeout)
            signal.alarm(seconds)
            try:
                result = func(*args, **kwargs)
            finally:
                signal.alarm(0)
            return result
        return wrapper
    return decorator


def extract_audio_features(file_path: str, quiet: bool = False, quick_mode: bool = False) -> Dict[str, Any]:
    """
    Extract comprehensive audio features from an audio file.
    
    Args:
        file_path: Path to the audio file
        quiet: If True, suppress progress messages (only output JSON)
        quick_mode: If True, analyze only first 60s with shorter timeout (for web UI)
        
    Returns:
        Dictionary containing all extracted features
    """
    def log(msg):
        if not quiet:
            print(msg, file=sys.stderr, flush=True)
    
    log(f"Analyzing: {file_path}")
    
    # Set duration and timeout based on mode
    duration = 60 if quick_mode else None  # None = full track
    timeout_seconds = 180 if quick_mode else 300  # 3 min for quick, 5 min for full
    
    # Load audio file with timeout protection
    @timeout(timeout_seconds)
    def load_audio():
        return librosa.load(file_path, sr=22050, duration=duration, mono=True)
    
    try:
        log(f"Loading audio file... ({'first 60s' if quick_mode else 'full track'})")
        y, sr = load_audio()
        log(f"Audio loaded: {len(y)/sr:.1f}s at {sr}Hz")
    except TimeoutError as e:
        return {"error": f"Audio loading timed out: {str(e)}"}
    except Exception as e:
        return {"error": f"Failed to load audio: {str(e)}"}
    
    features = {
        "file_path": file_path,
        "file_name": os.path.basename(file_path),
        "duration_seconds": float(librosa.get_duration(y=y, sr=sr))
    }
    
    # BPM (Tempo) - Use simpler onset-based method to avoid segfaults
    try:
        log("Extracting BPM...")
        # Use onset strength instead of full beat tracking (more stable)
        onset_env = librosa.onset.onset_strength(y=y, sr=sr)
        tempo = librosa.beat.tempo(onset_envelope=onset_env, sr=sr)[0]
        features["bpm"] = float(tempo)
        log(f"BPM: {tempo:.1f}")
    except Exception as e:
        log(f"BPM extraction failed: {e}")
        features["bpm"] = None
        features["bpm_error"] = str(e)
    
    # Key and mode detection (Chroma-based, Krumhansl-Kessler profiles)
    try:
        log("Detecting key and mode...")
        chroma = librosa.feature.chroma_cqt(y=y, sr=sr)
        chroma_mean = np.mean(chroma, axis=1)
        key_index = int(np.argmax(chroma_mean))
        keys = ['C', 'C#', 'D', 'D#', 'E', 'F', 'F#', 'G', 'G#', 'A', 'A#', 'B']
        features["key"] = keys[key_index]

        # Krumhansl-Kessler tonal hierarchy profiles
        major_profile = np.array([6.35, 2.23, 3.48, 2.33, 4.38, 4.09, 2.52, 5.19, 2.39, 3.66, 2.29, 2.88])
        minor_profile = np.array([6.33, 2.68, 3.52, 5.38, 2.60, 3.53, 2.54, 4.75, 3.98, 2.69, 3.34, 3.17])
        rotated_major = np.roll(major_profile, -key_index)
        rotated_minor = np.roll(minor_profile, -key_index)
        major_corr = float(np.corrcoef(chroma_mean, rotated_major)[0, 1])
        minor_corr = float(np.corrcoef(chroma_mean, rotated_minor)[0, 1])
        features["mode"] = "major" if major_corr > minor_corr else "minor"
        features["mode_confidence"] = float(abs(major_corr - minor_corr))
        log(f"Key: {features['key']} {features['mode']} (confidence {features['mode_confidence']:.3f})")
    except Exception as e:
        log(f"Key/mode detection failed: {e}")
        features["key"] = None
        features["mode"] = None
        features["key_error"] = str(e)
    
    # Spectral features
    try:
        log("Extracting spectral features...")
        spectral_centroids = librosa.feature.spectral_centroid(y=y, sr=sr)[0]
        features["spectral_centroid_mean"] = float(np.mean(spectral_centroids))
        features["spectral_centroid_std"] = float(np.std(spectral_centroids))
        
        spectral_rolloff = librosa.feature.spectral_rolloff(y=y, sr=sr)[0]
        features["spectral_rolloff_mean"] = float(np.mean(spectral_rolloff))
        log("Spectral features extracted")
    except Exception as e:
        log(f"Spectral features failed: {e}")
        features["spectral_error"] = str(e)
    
    # Zero crossing rate
    try:
        log("Calculating zero crossing rate...")
        zcr = librosa.feature.zero_crossing_rate(y)[0]
        features["zero_crossing_rate_mean"] = float(np.mean(zcr))
        log("ZCR calculated")
    except Exception as e:
        log(f"ZCR failed: {e}")
        features["zcr_error"] = str(e)
    
    # RMS Energy
    try:
        log("Calculating RMS energy...")
        rms = librosa.feature.rms(y=y)[0]
        features["rms_energy_mean"] = float(np.mean(rms))
        features["rms_energy_std"] = float(np.std(rms))
        # Loudness in dBFS (full scale) from RMS mean
        features["loudness_db"] = float(20 * np.log10(max(1e-8, features["rms_energy_mean"])))
        log(f"RMS energy calculated (loudness: {features['loudness_db']:.1f} dBFS)")
    except Exception as e:
        log(f"RMS energy failed: {e}")
        features["rms_error"] = str(e)
    
    # MFCC (Mel-frequency cepstral coefficients)
    try:
        log("Extracting MFCCs...")
        mfccs = librosa.feature.mfcc(y=y, sr=sr, n_mfcc=13)
        for i in range(13):
            features[f"mfcc_{i}_mean"] = float(np.mean(mfccs[i]))
            features[f"mfcc_{i}_std"] = float(np.std(mfccs[i]))
        log("MFCCs extracted")
    except Exception as e:
        log(f"MFCC extraction failed: {e}")
        features["mfcc_error"] = str(e)
    
    # High-level Spotify-like features (computed from low-level features)
    try:
        log("Computing high-level features...")
        
        # Energy: Normalized RMS energy (0-1)
        if "rms_energy_mean" in features:
            # Normalize RMS to 0-1 range (typical RMS is 0-0.5 for music)
            features["energy"] = float(min(1.0, features["rms_energy_mean"] * 2.5))
        else:
            features["energy"] = 0.0
        
        # Danceability: Based on tempo and beat strength
        if "bpm" in features and features["bpm"]:
            # Optimal dance BPM is 120-130, normalize around that
            bpm = features["bpm"]
            dance_score = 1.0 - abs(bpm - 125) / 125  # Peak at 125 BPM
            dance_score = max(0.0, min(1.0, dance_score * 1.2))  # Scale and clamp
            features["danceability"] = float(dance_score)
        else:
            features["danceability"] = 0.5
        
        # Valence: Based on spectral brightness and energy (mood/happiness)
        if "spectral_centroid_mean" in features:
            # Brighter sound (higher centroid) = more positive valence
            centroid_norm = min(1.0, features["spectral_centroid_mean"] / 4000)
            energy_factor = features.get("energy", 0.5)
            features["valence"] = float((centroid_norm * 0.6 + energy_factor * 0.4))
        else:
            features["valence"] = 0.5
        
        # Acousticness: Inverse of spectral rolloff (0-1)
        if "spectral_rolloff_mean" in features:
            # Lower rolloff = more acoustic
            rolloff_norm = min(1.0, features["spectral_rolloff_mean"] / 8000)
            features["acousticness"] = float(1.0 - rolloff_norm)
        else:
            features["acousticness"] = 0.5
        
        # Instrumentalness: Based on zero crossing rate
        # High ZCR suggests more instrumental content (less vocals)
        if "zero_crossing_rate_mean" in features:
            zcr_norm = min(1.0, features["zero_crossing_rate_mean"] * 5)
            features["instrumentalness"] = float(zcr_norm * 0.7)  # Scale down
        else:
            features["instrumentalness"] = 0.5
        
        # Speechiness: Inverse of instrumentalness with RMS variance
        if "rms_energy_std" in features:
            # High RMS variance suggests more speech-like dynamics
            rms_var_norm = min(1.0, features["rms_energy_std"] * 5)
            features["speechiness"] = float(rms_var_norm * 0.6)
        else:
            features["speechiness"] = 1.0 - features.get("instrumentalness", 0.5)
        
        # Liveness: Based on RMS energy variance (live = more dynamic)
        if "rms_energy_std" in features and "rms_energy_mean" in features:
            # High variance relative to mean = more live feel
            variance_ratio = features["rms_energy_std"] / max(0.01, features["rms_energy_mean"])
            features["liveness"] = float(min(1.0, variance_ratio * 0.8))
        else:
            features["liveness"] = 0.3
        
        # Melodicness: Based on chroma consistency (harmonic content)
        # Higher spectral centroid with lower variance = more melodic
        if "spectral_centroid_mean" in features and "spectral_centroid_std" in features:
            centroid_consistency = 1.0 - min(1.0, features["spectral_centroid_std"] / 2000)
            features["melodicness"] = float(centroid_consistency * 0.8)
        else:
            features["melodicness"] = 0.5
        
        log("High-level features computed")
    except Exception as e:
        log(f"High-level feature computation failed: {e}")
        # Set defaults if computation fails
        features["energy"] = 0.5
        features["danceability"] = 0.5
        features["valence"] = 0.5
        features["acousticness"] = 0.5
        features["instrumentalness"] = 0.5
        features["speechiness"] = 0.5
        features["liveness"] = 0.3
        features["melodicness"] = 0.5
    
    log("Analysis complete")
    return features



def generate_spectrogram(file_path: str, output_path: Optional[str] = None) -> Optional[str]:
    """
    Generate a mel spectrogram visualization for an audio file.
    
    Args:
        file_path: Path to the audio file
        output_path: Path to save the spectrogram image (optional)
        
    Returns:
        Path to the saved spectrogram image or None
    """
    try:
        y, sr = librosa.load(file_path, sr=None, duration=120)
        
        # Generate mel spectrogram
        S = librosa.feature.melspectrogram(y=y, sr=sr, n_mels=128, fmax=8000)
        S_dB = librosa.power_to_db(S, ref=np.max)
        
        # Create figure
        plt.figure(figsize=(12, 6))
        librosa.display.specshow(S_dB, sr=sr, x_axis='time', y_axis='mel', fmax=8000)
        plt.colorbar(format='%+2.0f dB')
        plt.title(f'Mel Spectrogram - {os.path.basename(file_path)}')
        plt.tight_layout()
        
        # Save or show
        if output_path:
            plt.savefig(output_path, dpi=150, bbox_inches='tight')
            print(f"Spectrogram saved: {output_path}")
            plt.close()
            return output_path
        else:
            plt.show()
            return None
            
    except Exception as e:
        print(f"Error generating spectrogram: {e}")
        return None


def analyze_directory(directory: str, output_json: str, generate_specs: bool = False):
    """
    Analyze all audio files in a directory.
    
    Args:
        directory: Directory containing audio files
        output_json: Path to save analysis results
        generate_specs: Whether to generate spectrogram images
    """
    results = []
    audio_extensions = {'.mp3', '.wav', '.flac', '.m4a', '.ogg', '.opus'}
    
    # Find all audio files
    audio_files = []
    for root, _, files in os.walk(directory):
        for file in files:
            if Path(file).suffix.lower() in audio_extensions:
                audio_files.append(os.path.join(root, file))
    
    print(f"Found {len(audio_files)} audio files to analyze")
    
    # Analyze each file
    for idx, file_path in enumerate(audio_files, 1):
        print(f"\n[{idx}/{len(audio_files)}] Processing...")
        
        features = extract_audio_features(file_path)
        results.append(features)
        
        # Generate spectrogram if requested
        if generate_specs and "error" not in features:
            spec_dir = os.path.join(directory, "spectrograms")
            os.makedirs(spec_dir, exist_ok=True)
            
            spec_filename = Path(file_path).stem + "_spectrogram.png"
            spec_path = os.path.join(spec_dir, spec_filename)
            
            spectrogram_path = generate_spectrogram(file_path, spec_path)
            if spectrogram_path:
                features["spectrogram_path"] = spectrogram_path
    
    # Save results
    with open(output_json, 'w') as f:
        json.dump(results, f, indent=2)
    
    print(f"\n✅ Analysis complete! Results saved to: {output_json}")
    print(f"   Analyzed {len(results)} files")
    print(f"   Errors: {sum(1 for r in results if 'error' in r)}")


def main():
    parser = argparse.ArgumentParser(description='Analyze audio files and extract metadata')
    parser.add_argument('input', help='Audio file or directory to analyze')
    parser.add_argument('-o', '--output', help='Output JSON file for results', 
                       default='audio_analysis.json')
    parser.add_argument('-s', '--spectrogram', action='store_true',
                       help='Generate spectrogram images')
    parser.add_argument('--single', action='store_true',
                       help='Analyze single file and print to stdout')
    parser.add_argument('-q', '--quiet', action='store_true',
                       help='Suppress progress messages (only output JSON)')
    parser.add_argument('--quick', action='store_true',
                       help='Quick mode: analyze first 60s only (faster, for web UI)')
    
    args = parser.parse_args()
    
    if not os.path.exists(args.input):
        print(f"Error: {args.input} does not exist", file=sys.stderr)
        sys.exit(1)
    
    if args.single or os.path.isfile(args.input):
        # Analyze single file
        features = extract_audio_features(args.input, quiet=args.quiet, quick_mode=args.quick)
        print(json.dumps(features, indent=2))
        
        if args.spectrogram:
            generate_spectrogram(args.input)
    else:
        # Analyze directory
        analyze_directory(args.input, args.output, args.spectrogram)


if __name__ == '__main__':
    main()
