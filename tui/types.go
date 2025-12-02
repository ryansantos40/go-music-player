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

type scanMsg struct {
	tracks []utils.Track
	err    error
}

type tickMsg time.Time

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
	focusedColumn   int
	cachedAlbumArt  string
	cachedTrackPath string
}
