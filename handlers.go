package main

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

func handleProfiles(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(store.ListProfiles())

		case http.MethodPost:
			var body struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
				http.Error(w, `{"error":"invalid name"}`, http.StatusBadRequest)
				return
			}
			if err := store.CreateProfile(strings.TrimSpace(body.Name)); err != nil {
				http.Error(w, `{"error":"save failed"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"name": strings.TrimSpace(body.Name)})

		case http.MethodDelete:
			name := r.URL.Query().Get("name")
			if name == "" {
				http.Error(w, `{"error":"missing name"}`, http.StatusBadRequest)
				return
			}
			if err := store.DeleteProfile(name); err != nil {
				http.Error(w, `{"error":"save failed"}`, http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handlePlaylists(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(store.ListPlaylists())

		case http.MethodPost:
			var body struct {
				URL string `json:"url"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
				http.Error(w, `{"error":"invalid url"}`, http.StatusBadRequest)
				return
			}
			// extract playlist ID from URL
			playlistID := extractPlaylistID(body.URL)
			if playlistID == "" {
				http.Error(w, `{"error":"could not extract playlist ID from URL"}`, http.StatusBadRequest)
				return
			}
			if err := store.AddPlaylist(playlistID, body.URL); err != nil {
				http.Error(w, `{"error":"save failed"}`, http.StatusInternalServerError)
				return
			}
			go fetchPlaylistVideos(store, playlistID, body.URL)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"id": playlistID})

		case http.MethodDelete:
			id := r.URL.Query().Get("id")
			if id == "" {
				http.Error(w, `{"error":"missing id"}`, http.StatusBadRequest)
				return
			}
			if err := store.DeletePlaylist(id); err != nil {
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
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, `{"error":"missing id"}`, http.StatusBadRequest)
			return
		}
		pl := store.GetPlaylist(id)
		if pl == nil {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pl)
	}
}

func handlePlaylistRefresh(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, `{"error":"missing id"}`, http.StatusBadRequest)
			return
		}
		pl := store.GetPlaylist(id)
		if pl == nil {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		store.SetPlaylistFetching(id, true)
		go fetchPlaylistVideos(store, id, pl.URL)
		w.WriteHeader(http.StatusAccepted)
	}
}

func handleProgress(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			profile := r.URL.Query().Get("profile")
			if profile == "" {
				http.Error(w, `{"error":"missing profile"}`, http.StatusBadRequest)
				return
			}
			pd := store.GetProgress(profile)
			if pd == nil {
				http.Error(w, `{"error":"profile not found"}`, http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(pd)

		case http.MethodPost:
			var body struct {
				Profile    string  `json:"profile"`
				PlaylistID string  `json:"playlist_id"`
				VideoID    string  `json:"video_id"`
				Timestamp  float64 `json:"timestamp"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Profile == "" || body.VideoID == "" {
				http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
				return
			}
			if err := store.SaveProgress(body.Profile, body.PlaylistID, body.VideoID, body.Timestamp); err != nil {
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
	// handle youtube.com/playlist?list=PLxxx
	if i := strings.Index(url, "list="); i != -1 {
		id := url[i+5:]
		if j := strings.IndexByte(id, '&'); j != -1 {
			id = id[:j]
		}
		return id
	}
	return ""
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
		store.SetPlaylistFetching(id, false)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("playlist fetch start error for %s: %v", id, err)
		store.SetPlaylistFetching(id, false)
		return
	}

	var videos []Video
	var playlistTitle string
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
		})
	}

	if err := cmd.Wait(); err != nil {
		log.Printf("playlist fetch error for %s: %v", id, err)
	}

	if playlistTitle == "" {
		playlistTitle = "Playlist " + id
	}

	log.Printf("fetched playlist %s: %d videos", id, len(videos))
	if err := store.SetPlaylistVideos(id, playlistTitle, videos); err != nil {
		log.Printf("playlist save error for %s: %v", id, err)
	}
}
