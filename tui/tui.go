package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ryansantos40/go-music-player/scanner"
)

type Model struct {
	viewport  viewport.Model
	textInput textinput.Model
	tracks    []scanner.Track
	scanning  bool
}

type scanMsg struct {
	tracks []scanner.Track
	err    error
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Enter directory to scan"
	ti.Focus()

	vp := viewport.New(80, 20)

	return Model{
		viewport:  vp,
		textInput: ti,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "enter":
			if !m.scanning {
				m.scanning = true
				return m, scanTracks(m.textInput.Value())
			}
		}

	case scanMsg:
		m.scanning = false
		if msg.err != nil {
			m.viewport.SetContent("Error scanning directory: " + msg.err.Error())

		} else {
			m.tracks = msg.tracks
			content := "Scanned Tracks:\n"
			for _, track := range m.tracks {
				content += fmt.Sprintf("- %s (%s)\n", track.Title, track.Artist)
			}
			m.viewport.SetContent(content)
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	m.viewport, _ = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString("Music Player TUI\n")
	b.WriteString("Type the directory and press <enter>\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	if m.scanning {
		b.WriteString("Scanning...\n")
	}

	b.WriteString(m.viewport.View())

	return b.String()
}

func scanTracks(dir string) tea.Cmd {
	return func() tea.Msg {
		tracks, err := scanner.ScanDir(dir)
		return scanMsg{tracks: tracks, err: err}
	}
}
