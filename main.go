package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

//go:embed index.html
var content embed.FS

// Shared client with aggressive connection reuse
var streamClient = &http.Client{
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		DisableCompression:    true,
	},
}

type AudioInfo struct {
	VideoID  string  `json:"video_id"`
	Title    string  `json:"title"`
	AudioURL string  `json:"audio_url"`
	Duration float64 `json:"duration"`
}

func main() {
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

func handleAudio(w http.ResponseWriter, r *http.Request) {
	ytURL := r.URL.Query().Get("url")
	if ytURL == "" {
		http.Error(w, `{"error":"missing url parameter"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "yt-dlp",
		"-f", "bestaudio[ext=m4a]/bestaudio[acodec=aac]/bestaudio",
		"--format-sort", "ext:m4a:mp4:webm",
		"--print", "%(duration)s",
		"--print", "%(title)s",
		"--print", "%(id)s",
		"--print", "%(url)s",
		"--no-playlist",
		ytURL,
	)

	out, err := cmd.Output()
	if err != nil {
		log.Printf("yt-dlp error: %v", err)
		http.Error(w, `{"error":"failed to extract audio"}`, http.StatusInternalServerError)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 4 {
		http.Error(w, `{"error":"unexpected yt-dlp output"}`, http.StatusInternalServerError)
		return
	}

	// --print outputs in order: duration, title, id, url
	dur, _ := strconv.ParseFloat(lines[0], 64)

	info := AudioInfo{
		Title:    lines[1],
		VideoID:  lines[2],
		AudioURL: lines[3],
		Duration: dur,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		http.Error(w, "missing url parameter", http.StatusBadRequest)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), "GET", rawURL, nil)
	if err != nil {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	if rng := r.Header.Get("Range"); rng != "" {
		req.Header.Set("Range", rng)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := streamClient.Do(req)
	if err != nil {
		log.Printf("stream fetch error: %v", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Flush data to browser as it arrives
	if flusher, ok := w.(http.Flusher); ok {
		buf := make([]byte, 32*1024)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				w.Write(buf[:n])
				flusher.Flush()
			}
			if err != nil {
				break
			}
		}
	} else {
		io.Copy(w, resp.Body)
	}
}

func init() {
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		fmt.Fprintln(os.Stderr, "WARNING: yt-dlp not found in PATH")
	}
}
