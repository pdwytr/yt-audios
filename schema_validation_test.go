package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDataDir_OK(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "users.csv"), "username,password\nadmin,admin1947\n")
	mustWriteFile(t, filepath.Join(dir, "channels.csv"), "id,name\nabc,Channel One\n")
	mustWriteFile(t, filepath.Join(dir, "playlists.csv"), "username,playlist_id,url,title\nadmin,PL123,https://youtube.com/playlist?list=PL123,Title\n")
	mustWriteFile(t, filepath.Join(dir, "videos.csv"), "playlist_id,video_id,title,duration,position\nPL123,vid1,Video One,12.5,0\n")
	mustWriteFile(t, filepath.Join(dir, "progress", "admin.json"), "{\n  \"playlist_id\": \"PL123\",\n  \"video_id\": \"vid1\",\n  \"timestamp\": 1.2,\n  \"positions\": {\n    \"PL123\": {\n      \"video_id\": \"vid1\",\n      \"timestamp\": 1.2\n    }\n  }\n}\n")

	if err := ValidateDataDir(dir); err != nil {
		t.Fatalf("expected valid data dir, got error: %v", err)
	}
}

func TestValidateDataDir_InvalidVideosHeader(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "videos.csv"), "playlist_id,video_id,title,duration\nPL123,vid1,Video One,12.5\n")

	err := ValidateDataDir(dir)
	if err == nil {
		t.Fatal("expected schema error for invalid videos header")
	}
}

func TestValidateDataDir_InvalidProgressTimestamp(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "progress", "admin.json"), "{\n  \"playlist_id\": \"PL123\",\n  \"video_id\": \"vid1\",\n  \"timestamp\": -1,\n  \"positions\": {}\n}\n")

	err := ValidateDataDir(dir)
	if err == nil {
		t.Fatal("expected schema error for invalid progress timestamp")
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
