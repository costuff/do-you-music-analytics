# test_tracks/

Generated test MP3 files live here (gitignored). Create them with:

```bash
bash scripts/generate-test-tracks.sh
```

Or manually:

```bash
ffmpeg -y -f lavfi -i "sine=frequency=440:duration=6" -ar 44100 -ac 2 -b:a 128k test-track-1.mp3
ffmpeg -y -f lavfi -i "sine=frequency=528:duration=8" -ar 44100 -ac 2 -b:a 128k test-track-2.mp3
ffmpeg -y -f lavfi -i "sine=frequency=320:duration=5" -ar 44100 -ac 2 -b:a 128k test-track-3.mp3
```

These are synthetic tone tracks (sine waves) that the librosa analyzer can process fully — good enough to test the full analysis pipeline locally without the NAS.
