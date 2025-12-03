package tui

import (
	"os"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/ryansantos40/go-music-player/utils"
)

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Enter music directory path..."
	ti.Focus()
	ti.Width = 60
	ti.PromptStyle = inputStyle
	ti.TextStyle = inputStyle
	ti.PlaceholderStyle = inputStyle

	vp := viewport.New(80, 20)

	playlistStore, _ := utils.NewPlaylistStore()

	cwd, _ := os.Getwd()
	fileExplorer := utils.NewFileExplorer(cwd)

	prog := progress.New(progress.WithScaledGradient(string(colorAccent), string(colorSuccess)),
		progress.WithoutPercentage(),
	)

	prog.Full = '▄'
	prog.FullColor = string(colorAccent)
	prog.Empty = '▄'
	prog.EmptyColor = string(colorSubtle)

	return Model{
		viewport:        vp,
		textInput:       ti,
		progressBar:     prog,
		selectedIndex:   0,
		mode:            ModeScan,
		lastTrackIdx:    -1,
		playlistStore:   playlistStore,
		currentPlaylist: "",
		inputMode:       InputNone,
		errorMsg:        "",
		width:           120,
		height:          30,
		fileExplorer:    fileExplorer,
		explorerIndex:   0,
		playlistIndex:   0,
		albumIndex:      0,
		artistIndex:     0,
		focusedColumn:   0,
		cachedAlbumArt:  "",
		cachedTrackPath: "",
		librarySection:  SectionPlaylists,
		currentFilter:   TrackFilter{Type: FilterAll, Key: "", Label: "All Tracks"},
	}
}
