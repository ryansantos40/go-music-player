package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ryansantos40/go-music-player/utils"
)

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.EnterAltScreen, tick())
}

func tick() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func scanTracks(dir string) tea.Cmd {
	return func() tea.Msg {
		tracks, err := utils.ScanDir(dir)
		return scanMsg{tracks: tracks, err: err}
	}
}

func (m Model) handleInputSubmit() Model {
	value := m.textInput.Value()

	switch m.inputMode {
	case InputPlaylistName:
		if err := m.playlistStore.CreatePlaylist(value); err != nil {
			m.errorMsg = err.Error()
		} else {
			m.currentPlaylist = value
		}

	case InputPlaylistLoad:
		if _, err := m.playlistStore.GetPlaylist(value); err != nil {
			m.errorMsg = err.Error()
		} else {
			m.currentPlaylist = value
		}
	}

	m.inputMode = InputNone
	m.textInput.Reset()
	return m
}

func (m Model) handleEnter() Model {
	if m.mode == ModeScan && !m.scanning {
		m.scanning = true
		return m
	}
	return m
}
