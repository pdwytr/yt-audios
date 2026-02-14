package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type PlaylistPosition struct {
	VideoID   string  `json:"video_id"`
	Timestamp float64 `json:"timestamp"`
}

type Profile struct {
	Name              string                       `json:"name"`
	GlobalPosition    *PlaylistPosition             `json:"global_position,omitempty"`
	PlaylistPositions map[string]*PlaylistPosition  `json:"playlist_positions"`
}

type Video struct {
	VideoID  string  `json:"video_id"`
	Title    string  `json:"title"`
	Duration float64 `json:"duration"`
}

type Playlist struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Videos    []Video   `json:"videos"`
	FetchedAt time.Time `json:"fetched_at"`
	Fetching  bool      `json:"fetching"`
}

type AppData struct {
	Profiles  map[string]*Profile  `json:"profiles"`
	Playlists map[string]*Playlist `json:"playlists"`
}

type Store struct {
	mu   sync.RWMutex
	data AppData
	path string
}

func NewStore(dir string) (*Store, error) {
	s := &Store{
		path: filepath.Join(dir, "data.json"),
		data: AppData{
			Profiles:  make(map[string]*Profile),
			Playlists: make(map[string]*Playlist),
		},
	}
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	f, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	var data AppData
	if err := json.Unmarshal(f, &data); err != nil {
		return err
	}
	if data.Profiles == nil {
		data.Profiles = make(map[string]*Profile)
	}
	if data.Playlists == nil {
		data.Playlists = make(map[string]*Playlist)
	}
	s.data = data
	return nil
}

func (s *Store) save() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}

// Profiles

func (s *Store) ListProfiles() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.data.Profiles))
	for name := range s.data.Profiles {
		names = append(names, name)
	}
	return names
}

func (s *Store) CreateProfile(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data.Profiles[name]; exists {
		return nil // already exists
	}
	s.data.Profiles[name] = &Profile{
		Name:              name,
		PlaylistPositions: make(map[string]*PlaylistPosition),
	}
	return s.save()
}

func (s *Store) DeleteProfile(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.Profiles, name)
	return s.save()
}

// Playlists

type PlaylistSummary struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	VideoCount int    `json:"video_count"`
	Fetching   bool   `json:"fetching"`
}

func (s *Store) ListPlaylists() []PlaylistSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]PlaylistSummary, 0, len(s.data.Playlists))
	for _, pl := range s.data.Playlists {
		out = append(out, PlaylistSummary{
			ID:         pl.ID,
			Title:      pl.Title,
			VideoCount: len(pl.Videos),
			Fetching:   pl.Fetching,
		})
	}
	return out
}

func (s *Store) AddPlaylist(id, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data.Playlists[id]; exists {
		return nil
	}
	s.data.Playlists[id] = &Playlist{
		ID:       id,
		URL:      url,
		Fetching: true,
	}
	return s.save()
}

func (s *Store) SetPlaylistFetching(id string, fetching bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pl, ok := s.data.Playlists[id]; ok {
		pl.Fetching = fetching
		return s.save()
	}
	return nil
}

func (s *Store) SetPlaylistVideos(id, title string, videos []Video) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pl, ok := s.data.Playlists[id]
	if !ok {
		return nil
	}
	pl.Title = title
	pl.Videos = videos
	pl.FetchedAt = time.Now()
	pl.Fetching = false
	return s.save()
}

func (s *Store) DeletePlaylist(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.Playlists, id)
	// clean up playlist positions from all profiles
	for _, p := range s.data.Profiles {
		delete(p.PlaylistPositions, id)
		if p.GlobalPosition != nil {
			// clear global position if it referred to this playlist
			// (we'd need a playlist_id on GlobalPosition for this — skip for now)
		}
	}
	return s.save()
}

func (s *Store) GetPlaylist(id string) *Playlist {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pl, ok := s.data.Playlists[id]
	if !ok {
		return nil
	}
	// return a copy
	cp := *pl
	cp.Videos = make([]Video, len(pl.Videos))
	copy(cp.Videos, pl.Videos)
	return &cp
}

// Progress

type ProgressData struct {
	PlaylistID string                       `json:"playlist_id,omitempty"`
	VideoID    string                       `json:"video_id,omitempty"`
	Timestamp  float64                      `json:"timestamp"`
	Positions  map[string]*PlaylistPosition `json:"positions,omitempty"`
}

func (s *Store) SaveProgress(profile, playlistID, videoID string, timestamp float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.data.Profiles[profile]
	if !ok {
		return nil
	}
	pos := &PlaylistPosition{VideoID: videoID, Timestamp: timestamp}
	p.GlobalPosition = &PlaylistPosition{VideoID: videoID, Timestamp: timestamp}
	if playlistID != "" {
		p.PlaylistPositions[playlistID] = pos
	}
	return s.save()
}

func (s *Store) GetProgress(profile string) *ProgressData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.data.Profiles[profile]
	if !ok {
		return nil
	}
	pd := &ProgressData{
		Positions: make(map[string]*PlaylistPosition),
	}
	if p.GlobalPosition != nil {
		pd.VideoID = p.GlobalPosition.VideoID
		pd.Timestamp = p.GlobalPosition.Timestamp
	}
	for k, v := range p.PlaylistPositions {
		cp := *v
		pd.Positions[k] = &cp
	}
	return pd
}
