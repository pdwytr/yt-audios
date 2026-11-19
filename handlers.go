package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	detected := false
	detectedAt := ""
	if t := botDetectedAt.Load(); t != nil {
		if time.Since(*t) < 30*time.Minute {
			detected = true
			detectedAt = t.Format(time.RFC3339)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"bot_detected":    detected,
		"bot_detected_at": detectedAt,
	})
}

func handleChannels(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(store.ListChannels())
	}
}

func handlePlaylists(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := getUser(r)

		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(store.ListUserPlaylists(username))

		case http.MethodPost:
			var body struct {
				URL string `json:"url"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
				http.Error(w, `{"error":"invalid url"}`, http.StatusBadRequest)
				return
			}
			playlistID := extractPlaylistID(body.URL)
			if playlistID == "" {
				http.Error(w, `{"error":"could not extract playlist ID from URL"}`, http.StatusBadRequest)
				return
			}

			// check playlist limit
			if len(store.ListUserPlaylists(username)) >= 3 {
				http.Error(w, `{"error":"playlist limit reached (max 3)"}`, http.StatusBadRequest)
				return
			}

			// validate channel whitelist
			vResult, err := validatePlaylistChannel(body.URL)
			if err != nil {
				log.Printf("channel validation error: %v", err)
				if err == errBotDetected {
					http.Error(w, `{"error":"bot_detected"}`, http.StatusServiceUnavailable)
					return
				}
				http.Error(w, `{"error":"failed to validate playlist channel"}`, http.StatusInternalServerError)
				return
			}
			if !store.IsChannelAllowed(vResult.ChannelID) {
				http.Error(w, `{"error":"channel not whitelisted"}`, http.StatusForbidden)
				return
			}

			if err := store.AddUserPlaylist(username, playlistID, body.URL, vResult.PlaylistTitle); err != nil {
				http.Error(w, `{"error":"save failed"}`, http.StatusInternalServerError)
				return
			}

			// fetch videos if not already cached
			if !store.HasVideosFor(playlistID) && !store.IsFetching(playlistID) {
				store.SetFetching(playlistID, true)
				go fetchPlaylistVideos(store, playlistID, body.URL)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"id": playlistID})

		case http.MethodDelete:
			id := r.URL.Query().Get("id")
			if id == "" {
				http.Error(w, `{"error":"missing id"}`, http.StatusBadRequest)
				return
			}
			if err := store.DeleteUserPlaylist(username, id); err != nil {
				http.Error(w, `{"error":"save failed"}`, http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handlePlaylistVideos(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		username := getUser(r)
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, `{"error":"missing id"}`, http.StatusBadRequest)
			return
		}
		if !store.UserHasPlaylist(username, id) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		videos := store.GetPlaylistVideos(id)
		_, fetching := store.fetchingPlaylists.Load(id)

		// find title from user's playlist entry
		playlists := store.ListUserPlaylists(username)
		title := id
		for _, pl := range playlists {
			if pl.ID == id {
				if pl.Title != "" {
					title = pl.Title
				}
				break
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       id,
			"title":    title,
			"videos":   videos,
			"fetching": fetching,
		})
	}
}

func handlePlaylistRefresh(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		username := getUser(r)
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, `{"error":"missing id"}`, http.StatusBadRequest)
			return
		}
		if !store.UserHasPlaylist(username, id) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		url := store.GetPlaylistURL(id)
		if url == "" {
			http.Error(w, `{"error":"playlist URL not found"}`, http.StatusNotFound)
			return
		}
		store.SetFetching(id, true)
		go fetchPlaylistVideos(store, id, url)
		w.WriteHeader(http.StatusAccepted)
	}
}

func handleProgress(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := getUser(r)

		switch r.Method {
		case http.MethodGet:
			pd := store.GetProgress(username)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(pd)

		case http.MethodPost:
			var body struct {
				PlaylistID string  `json:"playlist_id"`
				VideoID    string  `json:"video_id"`
				Timestamp  float64 `json:"timestamp"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.VideoID == "" {
				http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
				return
			}
			if err := store.SaveProgress(username, body.PlaylistID, body.VideoID, body.Timestamp); err != nil {
				http.Error(w, `{"error":"save failed"}`, http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func extractPlaylistID(url string) string {
	if i := strings.Index(url, "list="); i != -1 {
		id := url[i+5:]
		if j := strings.IndexByte(id, '&'); j != -1 {
			id = id[:j]
		}
		return id
	}
	return ""
}

var errBotDetected = fmt.Errorf("bot_detected")

type validationResult struct {
	ChannelID     string
	PlaylistTitle string
}

func validatePlaylistChannel(url string) (validationResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := append([]string{
		"--flat-playlist",
		"--playlist-items", "1",
		"--dump-json",
		"--no-download",
	}, cookieArgs()...)
	args = append(args, url)
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)

	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		if isBotDetection(stderr) {
			markBotDetected()
			return validationResult{}, errBotDetected
		}
		return validationResult{}, err
	}

	var entry struct {
		ChannelID     string `json:"channel_id"`
		PlaylistTitle string `json:"playlist_title"`
	}
	if err := json.Unmarshal(out, &entry); err != nil {
		return validationResult{}, err
	}
	return validationResult{ChannelID: entry.ChannelID, PlaylistTitle: entry.PlaylistTitle}, nil
}

func fetchPlaylistVideos(store *Store, id, url string) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	plArgs := append([]string{
		"--flat-playlist",
		"--dump-json",
		"--no-download",
	}, cookieArgs()...)
	plArgs = append(plArgs, url)
	cmd := exec.CommandContext(ctx, "yt-dlp", plArgs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("playlist fetch pipe error for %s: %v", id, err)
		store.SetFetching(id, false)
		return
	}

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		log.Printf("playlist fetch start error for %s: %v", id, err)
		store.SetFetching(id, false)
		return
	}

	var videos []Video
	var playlistTitle string
	position := 0
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)
	for scanner.Scan() {
		var entry struct {
			ID            string  `json:"id"`
			Title         string  `json:"title"`
			Duration      float64 `json:"duration"`
			PlaylistTitle string  `json:"playlist_title"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if playlistTitle == "" && entry.PlaylistTitle != "" {
			playlistTitle = entry.PlaylistTitle
		}
		videos = append(videos, Video{
			VideoID:  entry.ID,
			Title:    entry.Title,
			Duration: entry.Duration,
			Position: position,
		})
		position++
	}

	if err := cmd.Wait(); err != nil {
		log.Printf("playlist fetch error for %s: %v", id, err)
		if isBotDetection(stderrBuf.String()) {
			markBotDetected()
		}
	}

	// don't overwrite existing cache if fetch returned nothing
	if len(videos) == 0 {
		log.Printf("playlist fetch for %s returned 0 videos, keeping existing cache", id)
		store.SetFetching(id, false)
		return
	}

	if playlistTitle == "" {
		playlistTitle = "Playlist " + id
	}

	log.Printf("fetched playlist %s: %d videos", id, len(videos))
	if err := store.SetPlaylistVideos(id, playlistTitle, videos); err != nil {
		log.Printf("playlist save error for %s: %v", id, err)
	}
}
