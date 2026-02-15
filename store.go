package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

type Channel struct {
	ID   string `json:"channel_id"`
	Name string `json:"channel_name"`
}

type Video struct {
	VideoID  string  `json:"video_id"`
	Title    string  `json:"title"`
	Duration float64 `json:"duration"`
	Position int     `json:"position"`
}

type UserPlaylist struct {
	Username   string `json:"username"`
	PlaylistID string `json:"playlist_id"`
	URL        string `json:"url"`
	Title      string `json:"title"`
}

type PlaylistSummary struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	URL        string `json:"url"`
	VideoCount int    `json:"video_count"`
	Fetching   bool   `json:"fetching"`
}

type PlaylistPosition struct {
	VideoID   string  `json:"video_id"`
	Timestamp float64 `json:"timestamp"`
}

type ProgressData struct {
	PlaylistID string                       `json:"playlist_id,omitempty"`
	VideoID    string                       `json:"video_id,omitempty"`
	Timestamp  float64                      `json:"timestamp"`
	Positions  map[string]*PlaylistPosition `json:"positions,omitempty"`
}

type Store struct {
	mu               sync.RWMutex
	dataDir          string
	playlists        []UserPlaylist            // from playlists.csv
	videos           map[string][]Video        // playlistID -> videos, from videos.csv
	fetchingPlaylists sync.Map                 // playlistID -> chan struct{} (transient)
}

func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	progressDir := filepath.Join(dataDir, "progress")
	if err := os.MkdirAll(progressDir, 0755); err != nil {
		return nil, fmt.Errorf("create progress dir: %w", err)
	}

	s := &Store{
		dataDir: dataDir,
		videos:  make(map[string][]Video),
	}

	if err := s.loadPlaylists(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load playlists: %w", err)
	}
	if err := s.loadVideos(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load videos: %w", err)
	}
	return s, nil
}

// --- Channels ---

func (s *Store) LoadChannels() ([]Channel, error) {
	f, err := os.Open(filepath.Join(s.dataDir, "channels.csv"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	var channels []Channel
	for i, row := range rows {
		if i == 0 { // skip header
			continue
		}
		if len(row) >= 2 {
			channels = append(channels, Channel{ID: row[0], Name: row[1]})
		}
	}
	return channels, nil
}

func (s *Store) ListChannels() []Channel {
	channels, _ := s.LoadChannels()
	if channels == nil {
		return []Channel{}
	}
	return channels
}

func (s *Store) IsChannelAllowed(channelID string) bool {
	channels, err := s.LoadChannels()
	if err != nil {
		return false
	}
	for _, ch := range channels {
		if ch.ID == channelID {
			return true
		}
	}
	return false
}

// --- Playlists CSV ---

func (s *Store) playlistsPath() string {
	return filepath.Join(s.dataDir, "playlists.csv")
}

func (s *Store) loadPlaylists() error {
	f, err := os.Open(s.playlistsPath())
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return err
	}

	s.playlists = nil
	for i, row := range rows {
		if i == 0 { // skip header
			continue
		}
		if len(row) >= 3 {
			title := ""
			if len(row) >= 4 {
				title = row[3]
			}
			s.playlists = append(s.playlists, UserPlaylist{
				Username:   row[0],
				PlaylistID: row[1],
				URL:        row[2],
				Title:      title,
			})
		}
	}
	return nil
}

func (s *Store) savePlaylists() error {
	path := s.playlistsPath()
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	w.Write([]string{"username", "playlist_id", "url", "title"})
	for _, p := range s.playlists {
		w.Write([]string{p.Username, p.PlaylistID, p.URL, p.Title})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	f.Close()
	return os.Rename(tmp, path)
}

func (s *Store) AddUserPlaylist(username, playlistID, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// check if user already has this playlist
	for _, p := range s.playlists {
		if p.Username == username && p.PlaylistID == playlistID {
			return nil // already exists
		}
	}

	s.playlists = append(s.playlists, UserPlaylist{
		Username:   username,
		PlaylistID: playlistID,
		URL:        url,
	})
	return s.savePlaylists()
}

func (s *Store) ListUserPlaylists(username string) []PlaylistSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []PlaylistSummary
	for _, p := range s.playlists {
		if p.Username != username {
			continue
		}
		_, fetching := s.fetchingPlaylists.Load(p.PlaylistID)
		title := p.Title
		// if title is empty, check if videos have been fetched (title may be in another user's entry)
		if title == "" {
			for _, other := range s.playlists {
				if other.PlaylistID == p.PlaylistID && other.Title != "" {
					title = other.Title
					break
				}
			}
		}
		out = append(out, PlaylistSummary{
			ID:         p.PlaylistID,
			Title:      title,
			URL:        p.URL,
			VideoCount: len(s.videos[p.PlaylistID]),
			Fetching:   fetching,
		})
	}
	if out == nil {
		return []PlaylistSummary{}
	}
	return out
}

func (s *Store) DeleteUserPlaylist(username, playlistID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := s.playlists[:0]
	for _, p := range s.playlists {
		if p.Username == username && p.PlaylistID == playlistID {
			continue
		}
		filtered = append(filtered, p)
	}
	s.playlists = filtered
	return s.savePlaylists()
}

func (s *Store) UserHasPlaylist(username, playlistID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.playlists {
		if p.Username == username && p.PlaylistID == playlistID {
			return true
		}
	}
	return false
}

func (s *Store) GetPlaylistURL(playlistID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.playlists {
		if p.PlaylistID == playlistID {
			return p.URL
		}
	}
	return ""
}

func (s *Store) SetFetching(playlistID string, fetching bool) {
	if fetching {
		s.fetchingPlaylists.Store(playlistID, struct{}{})
	} else {
		s.fetchingPlaylists.Delete(playlistID)
	}
}

func (s *Store) IsFetching(playlistID string) bool {
	_, ok := s.fetchingPlaylists.Load(playlistID)
	return ok
}

// --- Videos CSV ---

func (s *Store) videosPath() string {
	return filepath.Join(s.dataDir, "videos.csv")
}

func (s *Store) loadVideos() error {
	f, err := os.Open(s.videosPath())
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return err
	}

	s.videos = make(map[string][]Video)
	for i, row := range rows {
		if i == 0 { // skip header
			continue
		}
		if len(row) >= 5 {
			dur, _ := strconv.ParseFloat(row[3], 64)
			pos, _ := strconv.Atoi(row[4])
			s.videos[row[0]] = append(s.videos[row[0]], Video{
				VideoID:  row[1],
				Title:    row[2],
				Duration: dur,
				Position: pos,
			})
		}
	}
	return nil
}

func (s *Store) saveVideos() error {
	path := s.videosPath()
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	w.Write([]string{"playlist_id", "video_id", "title", "duration", "position"})
	for playlistID, vids := range s.videos {
		for _, v := range vids {
			w.Write([]string{
				playlistID,
				v.VideoID,
				v.Title,
				strconv.FormatFloat(v.Duration, 'f', 2, 64),
				strconv.Itoa(v.Position),
			})
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	f.Close()
	return os.Rename(tmp, path)
}

func (s *Store) HasVideosFor(playlistID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.videos[playlistID]) > 0
}

func (s *Store) GetPlaylistVideos(playlistID string) []Video {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vids := s.videos[playlistID]
	out := make([]Video, len(vids))
	copy(out, vids)
	return out
}

func (s *Store) SetPlaylistVideos(playlistID, title string, videos []Video) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.videos[playlistID] = videos

	// update title on all playlist entries that reference this playlist
	if title != "" {
		for i := range s.playlists {
			if s.playlists[i].PlaylistID == playlistID && s.playlists[i].Title == "" {
				s.playlists[i].Title = title
			}
		}
		if err := s.savePlaylists(); err != nil {
			return err
		}
	}

	s.fetchingPlaylists.Delete(playlistID)
	return s.saveVideos()
}

// --- Progress (per-user JSON files) ---

func (s *Store) progressPath(username string) string {
	return filepath.Join(s.dataDir, "progress", username+".json")
}

func (s *Store) GetProgress(username string) *ProgressData {
	data, err := os.ReadFile(s.progressPath(username))
	if err != nil {
		return &ProgressData{Positions: make(map[string]*PlaylistPosition)}
	}
	var pd ProgressData
	if err := json.Unmarshal(data, &pd); err != nil {
		return &ProgressData{Positions: make(map[string]*PlaylistPosition)}
	}
	if pd.Positions == nil {
		pd.Positions = make(map[string]*PlaylistPosition)
	}
	return &pd
}

func (s *Store) SaveProgress(username, playlistID, videoID string, timestamp float64) error {
	pd := s.GetProgress(username)
	pd.VideoID = videoID
	pd.Timestamp = timestamp
	pd.PlaylistID = playlistID
	if pd.Positions == nil {
		pd.Positions = make(map[string]*PlaylistPosition)
	}
	if playlistID != "" {
		pd.Positions[playlistID] = &PlaylistPosition{
			VideoID:   videoID,
			Timestamp: timestamp,
		}
	}

	b, err := json.MarshalIndent(pd, "", "  ")
	if err != nil {
		return err
	}

	path := s.progressPath(username)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
