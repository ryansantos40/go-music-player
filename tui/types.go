package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/ryansantos40/go-music-player/utils"
)

type AppMode int
type InputMode int

type tickMsg time.Time

type LibrarySection int
type FilterType int

type scanMsg struct {
	tracks []utils.Track
	err    error
}

type TrackFilter struct {
	Type  FilterType
	Key   string
	Label string
}

type AlbumGroup struct {
	Key    string
	Title  string
	Artist string
	Year   int
	Tracks []utils.Track
}

type ArtistGroup struct {
	Key    string
	Name   string
	Tracks []utils.Track
}

const (
	FilterAll FilterType = iota
	FilterPlaylist
	FilterAlbum
	FilterArtist
)

const (
	SectionPlaylists LibrarySection = iota
	SectionAlbums
	SectionArtists
)

const (
	ModeExplorer AppMode = iota
	ModePlayer
	ModePlaylist
	ModeScan
)

const (
	InputNone InputMode = iota
	InputPlaylistName
	InputPlaylistLoad
)

type Model struct {
	viewport        viewport.Model
	textInput       textinput.Model
	progressBar     progress.Model
	tracks          []utils.Track
	scanning        bool
	player          *utils.Player
	selectedIndex   int
	mode            AppMode
	lastTrackIdx    int
	playlistStore   *utils.PlaylistStore
	currentPlaylist string
	inputMode       InputMode
	errorMsg        string
	width           int
	height          int
	fileExplorer    *utils.FileExplorer
	explorerIndex   int
	playlistIndex   int
	albumIndex      int
	artistIndex     int
	focusedColumn   int
	cachedAlbumArt  string
	cachedTrackPath string
	librarySection  LibrarySection
	currentFilter   TrackFilter
}
