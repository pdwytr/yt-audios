package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	usersCSVHeader     = []string{"username", "password"}
	channelsCSVHeader  = []string{"id", "name"}
	playlistsCSVHeader = []string{"username", "playlist_id", "url", "title"}
	videosCSVHeader    = []string{"playlist_id", "video_id", "title", "duration", "position"}
)

func ValidateDataDir(dataDir string) error {
	if err := validateUsersFile(filepath.Join(dataDir, "users.csv")); err != nil {
		return err
	}
	if err := validateChannelsFile(filepath.Join(dataDir, "channels.csv")); err != nil {
		return err
	}
	if err := validatePlaylistsFile(filepath.Join(dataDir, "playlists.csv")); err != nil {
		return err
	}
	if err := validateVideosFile(filepath.Join(dataDir, "videos.csv")); err != nil {
		return err
	}
	if err := validateProgressDir(filepath.Join(dataDir, "progress")); err != nil {
		return err
	}
	return nil
}

func validateUsersFile(path string) error {
	return validateCSVFile(path, usersCSVHeader, true, func(row []string, line int) error {
		if strings.TrimSpace(row[0]) == "" {
			return fmt.Errorf("line %d: username is empty", line)
		}
		if strings.TrimSpace(row[1]) == "" {
			return fmt.Errorf("line %d: password is empty", line)
		}
		return nil
	})
}

func validateChannelsFile(path string) error {
	return validateCSVFile(path, channelsCSVHeader, true, func(row []string, line int) error {
		if strings.TrimSpace(row[0]) == "" {
			return fmt.Errorf("line %d: channel id is empty", line)
		}
		if strings.TrimSpace(row[1]) == "" {
			return fmt.Errorf("line %d: channel name is empty", line)
		}
		return nil
	})
}

func validatePlaylistsFile(path string) error {
	return validateCSVFile(path, playlistsCSVHeader, true, func(row []string, line int) error {
		if strings.TrimSpace(row[0]) == "" {
			return fmt.Errorf("line %d: username is empty", line)
		}
		if strings.TrimSpace(row[1]) == "" {
			return fmt.Errorf("line %d: playlist_id is empty", line)
		}
		if strings.TrimSpace(row[2]) == "" {
			return fmt.Errorf("line %d: url is empty", line)
		}
		return nil
	})
}

func validateVideosFile(path string) error {
	return validateCSVFile(path, videosCSVHeader, true, func(row []string, line int) error {
		if strings.TrimSpace(row[0]) == "" {
			return fmt.Errorf("line %d: playlist_id is empty", line)
		}
		if strings.TrimSpace(row[1]) == "" {
			return fmt.Errorf("line %d: video_id is empty", line)
		}
		if strings.TrimSpace(row[2]) == "" {
			return fmt.Errorf("line %d: title is empty", line)
		}
		dur, err := strconv.ParseFloat(row[3], 64)
		if err != nil || math.IsNaN(dur) || math.IsInf(dur, 0) || dur < 0 {
			return fmt.Errorf("line %d: invalid duration %q", line, row[3])
		}
		pos, err := strconv.Atoi(row[4])
		if err != nil || pos < 0 {
			return fmt.Errorf("line %d: invalid position %q", line, row[4])
		}
		return nil
	})
}

func validateProgressDir(progressDir string) error {
	entries, err := os.ReadDir(progressDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read progress dir: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		if err := validateProgressFile(filepath.Join(progressDir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func validateProgressFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	var pd ProgressData
	if err := json.Unmarshal(data, &pd); err != nil {
		return fmt.Errorf("invalid json in %s: %w", path, err)
	}

	if math.IsNaN(pd.Timestamp) || math.IsInf(pd.Timestamp, 0) || pd.Timestamp < 0 {
		return fmt.Errorf("invalid timestamp in %s", path)
	}

	for playlistID, pos := range pd.Positions {
		if strings.TrimSpace(playlistID) == "" {
			return fmt.Errorf("invalid positions key in %s: empty playlist_id", path)
		}
		if pos == nil {
			return fmt.Errorf("invalid positions entry in %s for playlist %q: null value", path, playlistID)
		}
		if strings.TrimSpace(pos.VideoID) == "" {
			return fmt.Errorf("invalid positions entry in %s for playlist %q: empty video_id", path, playlistID)
		}
		if math.IsNaN(pos.Timestamp) || math.IsInf(pos.Timestamp, 0) || pos.Timestamp < 0 {
			return fmt.Errorf("invalid positions entry in %s for playlist %q: invalid timestamp", path, playlistID)
		}
	}

	return nil
}

func validateCSVFile(path string, expectedHeader []string, allowMissing bool, validateRow func(row []string, line int) error) error {
	f, err := os.Open(path)
	if err != nil {
		if allowMissing && os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if len(rows) == 0 {
		return fmt.Errorf("%s: missing header row", path)
	}
	if !rowsEqual(rows[0], expectedHeader) {
		return fmt.Errorf("%s: invalid header: got %q want %q", path, rows[0], expectedHeader)
	}

	for i, row := range rows[1:] {
		line := i + 2
		if len(row) != len(expectedHeader) {
			return fmt.Errorf("%s: line %d: expected %d columns, got %d", path, line, len(expectedHeader), len(row))
		}
		if validateRow != nil {
			if err := validateRow(row, line); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
		}
	}
	return nil
}

func rowsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.TrimSpace(a[i]) != b[i] {
			return false
		}
	}
	return true
}
