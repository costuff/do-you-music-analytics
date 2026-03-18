#!/usr/bin/env bash
# Generate synthetic test MP3 files in test_tracks/ for local development.
# Requires ffmpeg (brew install ffmpeg).
set -e
DIR="$(cd "$(dirname "$0")/.." && pwd)/test_tracks"
mkdir -p "$DIR"

generate() {
    local name="$1" freq="$2" dur="$3"
    local path="$DIR/$name"
    if [ -f "$path" ]; then
        echo "  skip $name (exists)"
    else
        ffmpeg -y -f lavfi -i "sine=frequency=${freq}:duration=${dur}" \
            -ar 44100 -ac 2 -b:a 128k "$path" 2>/dev/null
        echo "  ✅ $name"
    fi
}

echo "Generating test tracks in $DIR ..."
generate "test-track-1.mp3" 440 6
generate "test-track-2.mp3" 528 8
generate "test-track-3.mp3" 320 5
echo "Done. Set NAS_MOUNT_PATH=./test_tracks in .env.local to use these."
