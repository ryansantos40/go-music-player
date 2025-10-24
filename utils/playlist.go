package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Playlist struct {
	Name   string
	Tracks []Track
}

type PlaylistStore struct {
	configDir string
	playlists map[string]*Playlist
}

func NewPlaylistStore() (*PlaylistStore, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	ps := &PlaylistStore{
		configDir: configDir,
		playlists: make(map[string]*Playlist),
	}

	if err := os.MkdirAll(ps.getPlaylistDir(), 0755); err != nil {
		return nil, err
	}

	if err := ps.loadPlaylists(); err != nil {
		return nil, err
	}

	return ps, nil
}

func getConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "go-music-player"), nil
}

func (ps *PlaylistStore) getPlaylistDir() string {
	return filepath.Join(ps.configDir, "playlists")
}

func (ps *PlaylistStore) getPlaylistPath(name string) string {
	return filepath.Join(ps.getPlaylistDir(), name+".json")
}

func (ps *PlaylistStore) CreatePlaylist(name string) error {
	if _, exists := ps.playlists[name]; exists {
		return fmt.Errorf("playlist %s already exists", name)
	}

	ps.playlists[name] = &Playlist{
		Name:   name,
		Tracks: []Track{},
	}

	return ps.savePlaylist(name)
}

func (ps *PlaylistStore) DeletePlaylist(name string) error {
	if _, exists := ps.playlists[name]; !exists {
		return fmt.Errorf("playlist %s does not exist", name)
	}

	delete(ps.playlists, name)
	return os.Remove(ps.getPlaylistPath(name))
}

func (ps *PlaylistStore) AddTrack(playlistName string, track Track) error {
	playlist, exists := ps.playlists[playlistName]
	if !exists {
		return fmt.Errorf("playlist %s does not exist", playlistName)
	}

	for _, t := range playlist.Tracks {
		if t.Path == track.Path {
			return fmt.Errorf("track already exists in playlist")
		}
	}

	playlist.Tracks = append(playlist.Tracks, track)
	return ps.savePlaylist(playlistName)
}

func (ps *PlaylistStore) RemoveTrack(playlistName string, index int) error {
	playlist, exists := ps.playlists[playlistName]
	if !exists {
		return fmt.Errorf("playlist %s does not exist", playlistName)
	}

	if index < 0 || index >= len(playlist.Tracks) {
		return fmt.Errorf("track index out of range")
	}

	playlist.Tracks = append(playlist.Tracks[:index], playlist.Tracks[index+1:]...)
	return ps.savePlaylist(playlistName)
}

func (ps *PlaylistStore) GetPlaylist(name string) (*Playlist, error) {
	playlist, exists := ps.playlists[name]
	if !exists {
		return nil, fmt.Errorf("playlist %s does not exist", name)
	}
	return playlist, nil
}

func (ps *PlaylistStore) ListPlaylists() []string {
	names := make([]string, 0, len(ps.playlists))
	for name := range ps.playlists {
		names = append(names, name)
	}
	return names
}

func (ps *PlaylistStore) savePlaylist(name string) error {
	playlist, exists := ps.playlists[name]
	if !exists {
		return fmt.Errorf("playlist %s does not exist", name)
	}

	path := ps.getPlaylistPath(name)
	data, err := json.MarshalIndent(playlist, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (ps *PlaylistStore) loadPlaylists() error {
	dir := ps.getPlaylistDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			path := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			var playlist Playlist
			if err := json.Unmarshal(data, &playlist); err != nil {
				continue
			}

			ps.playlists[playlist.Name] = &playlist
		}
	}

	return nil
}

func (ps *PlaylistStore) ExportM3U(playlistName, exportPath string) error {
	playlist, err := ps.GetPlaylist(playlistName)
	if err != nil {
		return err
	}

	f, err := os.Create(exportPath)
	if err != nil {
		return err
	}
	defer f.Close()

	f.WriteString("#EXTM3U\n")
	for _, track := range playlist.Tracks {
		durationSeconds := int(track.Duration.Seconds())
		f.WriteString(fmt.Sprintf("#EXTINF:%d,%s - %s\n", durationSeconds, track.Artist, track.Title))
		f.WriteString(track.Path + "\n")
	}

	return nil
}
