package utils

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

type Track struct {
	Path     string
	Title    string
	Artist   string
	Album    string
	Duration time.Duration
	Year     int
}

var supportedExt = map[string]struct{}{
	".mp3":  {},
	".flac": {},
	".wav":  {},
	".ogg":  {},
}

func isAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := supportedExt[ext]
	return ok
}

func extractMetadata(path string) (Track, error) {
	file, err := os.Open(path)
	if err != nil {
		return Track{Path: path}, nil
	}
	defer file.Close()

	metadata, err := tag.ReadFrom(file)
	if err != nil {
		return Track{Path: path}, nil
	}

	track := Track{
		Path:   path,
		Title:  metadata.Title(),
		Artist: metadata.Artist(),
		Album:  metadata.Album(),
		Year:   metadata.Year(),
	}

	return track, nil
}

func ScanDir(root string) ([]Track, error) {
	var tracks []Track
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if isAudioFile(path) {
			track, _ := extractMetadata(path)
			tracks = append(tracks, track)
		}

		return nil
	})

	return tracks, err
}
