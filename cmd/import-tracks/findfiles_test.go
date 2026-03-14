package main

import (
	"os"
	"path/filepath"
	"testing"
)

func mkfile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("fake"), 0644); err != nil {
		t.Fatalf("mkfile %s: %v", path, err)
	}
}

func TestFindMP3Files_Shallow_TopLevelOnly(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	mkfile(t, filepath.Join(dir, "a.mp3"))
	mkfile(t, filepath.Join(dir, "b.mp3"))
	mkfile(t, filepath.Join(sub, "nested.mp3"))

	files, err := findMP3Files(dir, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("shallow: want 2 top-level files, got %d", len(files))
	}
}

func TestFindMP3Files_Deep_FindsNested(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "artist", "album")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	mkfile(t, filepath.Join(dir, "a.mp3"))
	mkfile(t, filepath.Join(sub, "b.mp3"))
	mkfile(t, filepath.Join(sub, "c.mp3"))

	files, err := findMP3Files(dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 3 {
		t.Errorf("deep: want 3 nested files, got %d", len(files))
	}
}

func TestFindMP3Files_IgnoresHiddenFiles(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, filepath.Join(dir, "track.mp3"))
	mkfile(t, filepath.Join(dir, ".hidden.mp3"))
	mkfile(t, filepath.Join(dir, "._resource.mp3"))

	for _, shallow := range []bool{true, false} {
		files, err := findMP3Files(dir, shallow)
		if err != nil {
			t.Fatalf("shallow=%v: %v", shallow, err)
		}
		if len(files) != 1 {
			t.Errorf("shallow=%v: want 1 visible file, got %d", shallow, len(files))
		}
	}
}

func TestFindMP3Files_IgnoresNonMP3(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, filepath.Join(dir, "track.mp3"))
	mkfile(t, filepath.Join(dir, "cover.jpg"))
	mkfile(t, filepath.Join(dir, "notes.txt"))

	for _, shallow := range []bool{true, false} {
		files, err := findMP3Files(dir, shallow)
		if err != nil {
			t.Fatalf("shallow=%v: %v", shallow, err)
		}
		if len(files) != 1 {
			t.Errorf("shallow=%v: want 1 MP3, got %d", shallow, len(files))
		}
	}
}

func TestFindMP3Files_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	for _, shallow := range []bool{true, false} {
		files, err := findMP3Files(dir, shallow)
		if err != nil {
			t.Fatalf("shallow=%v: %v", shallow, err)
		}
		if len(files) != 0 {
			t.Errorf("shallow=%v: want 0, got %d", shallow, len(files))
		}
	}
}

func TestFindMP3Files_NonExistentDir(t *testing.T) {
	for _, shallow := range []bool{true, false} {
		if _, err := findMP3Files("/tmp/nope_does_not_exist_xyz", shallow); err == nil {
			t.Errorf("shallow=%v: want error for non-existent dir, got nil", shallow)
		}
	}
}

func TestFindMP3Files_CaseInsensitiveExtension(t *testing.T) {
	dir := t.TempDir()
	mkfile(t, filepath.Join(dir, "a.mp3"))
	mkfile(t, filepath.Join(dir, "b.MP3"))
	mkfile(t, filepath.Join(dir, "c.Mp3"))

	for _, shallow := range []bool{true, false} {
		files, err := findMP3Files(dir, shallow)
		if err != nil {
			t.Fatalf("shallow=%v: %v", shallow, err)
		}
		if len(files) != 3 {
			t.Errorf("shallow=%v: want 3 case-insensitive matches, got %d", shallow, len(files))
		}
	}
}

func TestFindMP3Files_ShallowVsDeepCount(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	mkfile(t, filepath.Join(dir, "top.mp3"))
	mkfile(t, filepath.Join(sub, "nested1.mp3"))
	mkfile(t, filepath.Join(sub, "nested2.mp3"))

	shallow, err := findMP3Files(dir, true)
	if err != nil {
		t.Fatalf("shallow: %v", err)
	}
	deep, err := findMP3Files(dir, false)
	if err != nil {
		t.Fatalf("deep: %v", err)
	}
	if len(shallow) != 1 {
		t.Errorf("shallow: want 1, got %d", len(shallow))
	}
	if len(deep) != 3 {
		t.Errorf("deep: want 3, got %d", len(deep))
	}
	if len(shallow) >= len(deep) {
		t.Errorf("shallow (%d) should find fewer files than deep (%d)", len(shallow), len(deep))
	}
}
