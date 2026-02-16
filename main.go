package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//go:embed all:frontend/dist
var distFS embed.FS

// tracks in-progress downloads: videoID -> channel that closes when done
var downloads sync.Map

// directory for downloaded audio files (persists across restarts)
var audioDir string

// botDetectedAt records the last time YouTube bot detection was seen
var botDetectedAt atomic.Pointer[time.Time]

// isBotDetection checks if yt-dlp output contains bot detection signals
func isBotDetection(output string) bool {
	patterns := []string{
		"Sign in to confirm you're not a bot",
		"Sign in to confirm your age",
		"HTTP Error 429",
		"This request was detected as a bot",
	}
	for _, p := range patterns {
		if strings.Contains(output, p) {
			return true
		}
	}
	return false
}

func markBotDetected() {
	now := time.Now()
	botDetectedAt.Store(&now)
	log.Println("[BOT DETECTED] YouTube is blocking requests — cookies may need to be refreshed")
}

type AudioInfo struct {
	VideoID  string  `json:"video_id"`
	Title    string  `json:"title"`
	Duration float64 `json:"duration"`
}

func main() {
	// init data store (DATA_DIR env, or ./data in CWD)
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}

	// audio cache directory (persists across restarts)
	audioDir = filepath.Join(dataDir, "audio")
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		log.Fatal(err)
	}
	store, err := NewStore(dataDir)
	if err != nil {
		log.Fatalf("failed to init store: %v", err)
	}

	auth := NewAuth(filepath.Join(dataDir, "users.csv"))

	// fetch missing suggestion titles in background
	go fetchSuggestionTitles(store)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	distSub, err := fs.Sub(distFS, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/", spaHandler(http.FS(distSub)))
	http.HandleFunc("/audio", handleAudio)
	http.HandleFunc("/stream", handleStream)

	// API routes
	http.HandleFunc("/api/login", handleLogin(auth))
	http.HandleFunc("/api/register", handleRegister(auth))
	http.HandleFunc("/api/register/info", handleRegisterInfo(auth))
	http.HandleFunc("/api/channels", handleChannels(store))
	http.HandleFunc("/api/playlists/suggestions", handlePlaylistSuggestions(store))
	http.HandleFunc("/api/playlists", auth.middleware(handlePlaylists(store)))
	http.HandleFunc("/api/playlists/videos", auth.middleware(handlePlaylistVideos(store)))
	http.HandleFunc("/api/playlists/refresh", auth.middleware(handlePlaylistRefresh(store)))
	http.HandleFunc("/api/progress", auth.middleware(handleProgress(store)))
	http.HandleFunc("/api/status", handleStatus)

	log.Printf("Listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func audioPath(videoID string) string {
	return filepath.Join(audioDir, videoID+".m4a")
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
	args := append([]string{
		"-f", "bestaudio[ext=m4a]/bestaudio[acodec=aac]/bestaudio",
		"--format-sort", "ext:m4a:mp4:webm",
		"--print", "%(duration)s",
		"--print", "%(title)s",
		"--print", "%(id)s",
		"--no-playlist",
	}, cookieArgs()...)
	args = append(args, ytURL)
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)

	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		log.Printf("yt-dlp error: %v | stderr: %s", err, stderr)
		if isBotDetection(stderr) {
			markBotDetected()
			http.Error(w, `{"error":"bot_detected"}`, http.StatusServiceUnavailable)
			return
		}
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

	dlArgs := append([]string{
		"-f", "bestaudio[ext=m4a]/bestaudio[acodec=aac]/bestaudio",
		"--format-sort", "ext:m4a:mp4:webm",
		"-o", filePath,
		"--no-part",
		"--no-playlist",
	}, cookieArgs()...)
	dlArgs = append(dlArgs, ytURL)
	cmd := exec.CommandContext(ctx, "yt-dlp", dlArgs...)

	start := time.Now()
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("download error for %s: %v | output: %s", videoID, err, string(out))
		if isBotDetection(string(out)) {
			markBotDetected()
		}
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

// cookieArgs returns yt-dlp cookie flags.
// YT_COOKIES=cookies.txt  → --cookies cookies.txt
// YT_BROWSER=chrome        → --cookies-from-browser chrome
// Both unset               → no cookie args
func cookieArgs() []string {
	if f := os.Getenv("YT_COOKIES"); f != "" {
		return []string{"--cookies", f}
	}
	if b := os.Getenv("YT_BROWSER"); b != "" && b != "none" {
		return []string{"--cookies-from-browser", b}
	}
	return nil
}

func init() {
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		fmt.Fprintln(os.Stderr, "WARNING: yt-dlp not found in PATH")
	}
}

// spaHandler serves static files from the embedded FS, falling back to
// index.html for any path that doesn't match a real file (SPA client routing).
func spaHandler(fsys http.FileSystem) http.Handler {
	fileServer := http.FileServer(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// try to open the file; if it exists, serve it
		f, err := fsys.Open(r.URL.Path)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		// file not found — serve index.html for client-side routing
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
