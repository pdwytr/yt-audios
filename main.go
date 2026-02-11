package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed index.html
var content embed.FS

// tracks in-progress downloads: videoID -> channel that closes when done
var downloads sync.Map

// temp directory for downloaded audio files
var tmpDir string

type AudioInfo struct {
	VideoID  string  `json:"video_id"`
	Title    string  `json:"title"`
	Duration float64 `json:"duration"`
}

func main() {
	// create a temp directory for audio files
	var err error
	tmpDir, err = os.MkdirTemp("", "yt-audios-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/audio", handleAudio)
	http.HandleFunc("/stream", handleStream)

	log.Printf("Listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, _ := content.ReadFile("index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func audioPath(videoID string) string {
	return filepath.Join(tmpDir, videoID+".m4a")
}

func handleAudio(w http.ResponseWriter, r *http.Request) {
	ytURL := r.URL.Query().Get("url")
	if ytURL == "" {
		http.Error(w, `{"error":"missing url parameter"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// get metadata
	cmd := exec.CommandContext(ctx, "yt-dlp",
		"-f", "bestaudio[ext=m4a]/bestaudio[acodec=aac]/bestaudio",
		"--format-sort", "ext:m4a:mp4:webm",
		"--print", "%(duration)s",
		"--print", "%(title)s",
		"--print", "%(id)s",
		"--no-playlist",
		ytURL,
	)

	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		log.Printf("yt-dlp error: %v | stderr: %s", err, stderr)
		http.Error(w, `{"error":"failed to extract audio"}`, http.StatusInternalServerError)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 3 {
		http.Error(w, `{"error":"unexpected yt-dlp output"}`, http.StatusInternalServerError)
		return
	}

	dur, _ := strconv.ParseFloat(lines[0], 64)
	videoID := lines[2]

	info := AudioInfo{
		Title:    lines[1],
		VideoID:  videoID,
		Duration: dur,
	}

	// start background download if not already cached or in progress
	filePath := audioPath(videoID)
	if _, err := os.Stat(filePath); err != nil {
		// file doesn't exist yet — check if download is already running
		if _, loaded := downloads.LoadOrStore(videoID, make(chan struct{})); !loaded {
			go downloadAudio(videoID, ytURL)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func downloadAudio(videoID, ytURL string) {
	ch, _ := downloads.Load(videoID)
	defer close(ch.(chan struct{}))

	filePath := audioPath(videoID)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "yt-dlp",
		"-f", "bestaudio[ext=m4a]/bestaudio[acodec=aac]/bestaudio",
		"--format-sort", "ext:m4a:mp4:webm",
		"-o", filePath,
		"--no-part",
		"--no-playlist",
		ytURL,
	)

	start := time.Now()
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("download error for %s: %v | output: %s", videoID, err, string(out))
		os.Remove(filePath)
		return
	}
	log.Printf("downloaded %s in %s", videoID, time.Since(start).Round(time.Millisecond))
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	videoID := r.URL.Query().Get("id")
	if videoID == "" {
		http.Error(w, "missing id parameter", http.StatusBadRequest)
		return
	}

	filePath := audioPath(videoID)

	// if download is in progress, wait for it
	if ch, ok := downloads.Load(videoID); ok {
		select {
		case <-ch.(chan struct{}):
			// download finished
		case <-r.Context().Done():
			return
		}
	}

	// check file exists
	if _, err := os.Stat(filePath); err != nil {
		http.Error(w, `{"error":"audio not found"}`, http.StatusNotFound)
		return
	}

	// http.ServeFile handles Range requests, Content-Length, etc. automatically
	http.ServeFile(w, r, filePath)
}

func init() {
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		fmt.Fprintln(os.Stderr, "WARNING: yt-dlp not found in PATH")
	}
}
