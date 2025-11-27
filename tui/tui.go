package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ryansantos40/go-music-player/utils"
)

var (
	colorBg      = lipgloss.Color("#1e2139")
	colorFg      = lipgloss.Color("#e0e0e0")
	colorAccent  = lipgloss.Color("#6ee7b7")
	colorSuccess = lipgloss.Color("#6ee7b7")
	colorDanger  = lipgloss.Color("#FF0000")
	colorSubtle  = lipgloss.Color("#9ca3b0")

	appStyle = lipgloss.NewStyle().
			Background(colorBg).
			Foreground(colorFg)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorFg).
			Background(colorBg)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorFg).
			Background(colorBg).
			Align(lipgloss.Center)

	sectionTitleStyle = lipgloss.NewStyle().
				Foreground(colorFg).
				Background(colorBg).
				Bold(true).
				Align(lipgloss.Center)

	statusStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Background(colorBg)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorDanger).
			Background(colorBg).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Background(colorBg).
			Bold(true)

	playingStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Background(colorBg).
			Bold(true)

	subtleStyle = lipgloss.NewStyle().
			Foreground(colorSubtle).
			Background(colorBg)

	inputStyle = lipgloss.NewStyle().
			Background(colorBg).
			Foreground(colorFg)

	borderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorSubtle).
			Background(colorBg)

	columnStyle = lipgloss.NewStyle().
			Background(colorBg).
			Padding(0, 1)
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
	focusedColumn   int // 0=playlists, 1=tracks
	cachedAlbumArt  string
	cachedTrackPath string
}

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

	prog.Full = '●' //▄
	prog.FullColor = string(colorAccent)
	prog.Empty = '○' //▄
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
		focusedColumn:   0,
		cachedAlbumArt:  "",
		cachedTrackPath: "",
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.EnterAltScreen, tick())
}

func tick() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || (msg.String() == "q" && m.inputMode == InputNone && m.mode != ModeScan) {
			if m.player != nil {
				m.player.Stop()
			}
			return m, tea.Quit
		}

		if msg.String() == "esc" {
			if m.mode == ModeExplorer {
				m.mode = ModeScan
				m.explorerIndex = 0
				return m, nil
			}
			m.inputMode = InputNone
			m.textInput.Reset()
			m.errorMsg = ""
			return m, nil
		}

		// Input modes
		if m.inputMode != InputNone || m.mode == ModeScan {
			if msg.String() == "enter" {
				if m.inputMode != InputNone {
					m = m.handleInputSubmit()
					return m, nil
				}

				if m.mode == ModeScan {
					m = m.handleEnter()
					return m, scanTracks(m.textInput.Value())
				}
			}

			if m.mode == ModeScan && msg.String() == "tab" {
				m.mode = ModeExplorer
				m.explorerIndex = 0
				return m, nil
			}

			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		// Explorer mode
		if m.mode == ModeExplorer {
			switch msg.String() {
			case "up", "k":
				if m.explorerIndex > 0 {
					m.explorerIndex--
				}
			case "down", "j":
				if m.explorerIndex < len(m.fileExplorer.Entries)-1 {
					m.explorerIndex++
				}
			case " ":
				m.fileExplorer.EnterDirectory(m.explorerIndex)
				m.explorerIndex = 0
			case "enter":
				selectedPath := m.fileExplorer.GetCurrentPath()
				m.scanning = true
				return m, scanTracks(selectedPath)
			case "backspace", "h":
				m.fileExplorer.GoToParent()
				m.explorerIndex = 0
			}
			return m, nil
		}

		// Main player navigation
		if m.mode == ModePlayer {
			switch msg.String() {
			case "tab":
				m.focusedColumn = (m.focusedColumn + 1) % 2 // 0=playlists, 1=tracks

			case "up", "k":
				if m.focusedColumn == 0 {
					// Navigate playlists
					playlists := m.playlistStore.ListPlaylists()
					if len(playlists) > 0 {
						m.playlistIndex = (m.playlistIndex - 1 + len(playlists)) % len(playlists)
						m.currentPlaylist = playlists[m.playlistIndex]
						m.selectedIndex = 0
					}
				} else if m.focusedColumn == 1 {
					// Navigate tracks
					tracksLen := len(m.tracks)
					if m.currentPlaylist != "" {
						if playlist, err := m.playlistStore.GetPlaylist(m.currentPlaylist); err == nil {
							tracksLen = len(playlist.Tracks)
						}
					}
					if tracksLen > 0 {
						m.selectedIndex = (m.selectedIndex - 1 + tracksLen) % tracksLen
					}
				}

			case "down", "j":
				if m.focusedColumn == 0 {
					playlists := m.playlistStore.ListPlaylists()
					if len(playlists) > 0 {
						m.playlistIndex = (m.playlistIndex + 1) % len(playlists)
						m.currentPlaylist = playlists[m.playlistIndex]
						m.selectedIndex = 0
					}
				} else if m.focusedColumn == 1 {
					tracksLen := len(m.tracks)
					if m.currentPlaylist != "" {
						if playlist, err := m.playlistStore.GetPlaylist(m.currentPlaylist); err == nil {
							tracksLen = len(playlist.Tracks)
						}
					}
					if tracksLen > 0 {
						m.selectedIndex = (m.selectedIndex + 1) % tracksLen
					}
				}

			case "enter":
				if m.focusedColumn == 1 && m.player != nil {
					if m.currentPlaylist != "" {
						if playlist, err := m.playlistStore.GetPlaylist(m.currentPlaylist); err == nil {
							if m.selectedIndex < len(playlist.Tracks) {
								m.player = utils.NewPlayer(playlist.Tracks)
								m.player.Skip(m.selectedIndex)
								m.lastTrackIdx = m.selectedIndex
							}
						}
					} else if len(m.tracks) > 0 {
						m.player.Skip(m.selectedIndex)
						m.lastTrackIdx = m.selectedIndex
					}
				}

			case " ":
				if m.player != nil {
					if m.player.IsPlaying() {
						m.player.Pause()
					} else {
						m.player.Resume()
					}
				}

			case "n":
				if m.player != nil {
					m.player.Next()
				}

			case "p":
				if m.player != nil {
					m.player.Previous()
				}

			case "s":
				if m.player != nil {
					m.player.ToggleShuffle()
				}

			case "r":
				if m.player != nil {
					m.player.ToggleRepeat()
				}

			case "right", "l":
				if m.player != nil && m.player.IsPlaying() {
					m.player.SeekForward()
					return m, tick()
				}

			case "left", "h":
				if m.player != nil && m.player.IsPlaying() {
					m.player.SeekBackward()
					return m, tick()
				}

			case "+", "=":
				if m.player != nil {
					m.player.SetVolume(m.player.GetVolume() + 0.1)
				}

			case "-", "_":
				if m.player != nil {
					m.player.SetVolume(m.player.GetVolume() - 0.1)
				}

			case "c":
				m.inputMode = InputPlaylistName
				m.textInput.Placeholder = "Enter playlist name..."
				m.textInput.Focus()

			case "a":
				if m.currentPlaylist != "" && m.player != nil {
					track := m.player.GetCurrentTrack()
					if err := m.playlistStore.AddTrack(m.currentPlaylist, track); err != nil {
						m.errorMsg = err.Error()
					}
				}

			case "x":
				if m.currentPlaylist != "" && m.focusedColumn == 1 {
					if err := m.playlistStore.RemoveTrack(m.currentPlaylist, m.selectedIndex); err != nil {
						m.errorMsg = err.Error()
					}
				}

			case "d":
				if m.currentPlaylist != "" && m.focusedColumn == 0 {
					if err := m.playlistStore.DeletePlaylist(m.currentPlaylist); err != nil {
						m.errorMsg = err.Error()
					} else {
						m.currentPlaylist = ""
						m.playlistIndex = 0
					}
				}
			}
		}

	case scanMsg:
		m.scanning = false
		if msg.err != nil {
			m.errorMsg = "Error: " + msg.err.Error()
			m.mode = ModeExplorer
		} else {
			m.tracks = msg.tracks
			m.mode = ModePlayer
			m.player = utils.NewPlayer(m.tracks)
			m.player.Play()
			m.lastTrackIdx = m.player.GetCurrentIndex()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 10
		m.progressBar.Width = msg.Width - 2

	case tickMsg:
		if m.mode == ModePlayer && m.player != nil && m.player.IsPlaying() {
			currentIdx := m.player.GetCurrentIndex()
			if currentIdx != m.lastTrackIdx {
				m.lastTrackIdx = currentIdx
			}
		}
		cmd = tick()
		cmds = append(cmds, cmd)

	case progress.FrameMsg:
		progressModel, cmd := m.progressBar.Update(msg)
		m.progressBar = progressModel.(progress.Model)
		cmds = append(cmds, cmd)

	}

	return m, tea.Batch(cmds...)
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

func (m Model) View() string {
	if m.inputMode != InputNone {
		return appStyle.Width(m.width).Height(m.height).Render(m.renderInput())
	}

	switch m.mode {
	case ModeScan:
		return appStyle.Width(m.width).Height(m.height).Render(m.renderScanMode())
	case ModeExplorer:
		return appStyle.Width(m.width).Height(m.height).Render(m.renderExplorerMode())
	case ModePlayer:
		return appStyle.Width(m.width).Height(m.height).Render(m.renderPlayerMode())
	}

	return ""
}

func (m Model) renderPlayerMode() string {
	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Progress bar
	b.WriteString(m.renderProgressBar())
	b.WriteString("\n")

	// 3 Columns layout com bordas compartilhadas
	b.WriteString(m.renderColumns())

	b.WriteString("\n")

	// Footer with commands
	b.WriteString(m.renderCommands())

	return b.String()
}

func (m Model) renderColumns() string {
	var b strings.Builder

	col1Width := m.width / 3
	col2Width := m.width / 3
	col3Width := m.width - col1Width - col2Width

	col1ContentWidth := col1Width - 2
	col2ContentWidth := col2Width - 2
	col3ContentWidth := col3Width - 2

	colHeight := m.height - 14

	col1Lines := m.getColumnLines(m.renderPlaylistColumn(), colHeight)
	col2Lines := m.getColumnLines(m.renderTracksColumn(), colHeight)
	col3Lines := m.getColumnLines(m.renderVisualizerColumn(), colHeight)

	topBorder := "┌" + strings.Repeat("─", col1ContentWidth) + "┬" + strings.Repeat("─", col2ContentWidth) + "┬" + strings.Repeat("─", col3ContentWidth) + "┐"
	b.WriteString(topBorder + "\n")

	for i := 0; i < colHeight; i++ {
		line1 := m.padOrTruncate(col1Lines[i], col1ContentWidth)
		line2 := m.padOrTruncate(col2Lines[i], col2ContentWidth)
		line3 := m.padOrTruncate(col3Lines[i], col3ContentWidth)

		row := "│" + line1 + "│" + line2 + "│" + line3 + "│"
		b.WriteString(row + "\n")
	}

	bottomBorder := "└" + strings.Repeat("─", col1ContentWidth) + "┴" + strings.Repeat("─", col2ContentWidth) + "┴" + strings.Repeat("─", col3ContentWidth) + "┘"
	b.WriteString(bottomBorder)

	return b.String()
}

func (m Model) getColumnLines(content string, height int) []string {
	lines := strings.Split(content, "\n")
	result := make([]string, height)

	for i := 0; i < height; i++ {
		if i < len(lines) {
			result[i] = lines[i]
		} else {
			result[i] = ""
		}
	}

	return result
}

func (m Model) padOrTruncate(s string, width int) string {
	visualWidth := lipgloss.Width(s)

	if visualWidth >= width {
		return m.truncate(s, width)
	}

	padding := width - visualWidth
	return s + strings.Repeat(" ", padding)
}

func (m Model) truncate(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	runes := []rune(s)
	result := ""
	for _, r := range runes {
		test := result + string(r)
		if lipgloss.Width(test) > maxWidth-3 {
			return result + "..."
		}
		result = test
	}
	return result
}

func (m Model) renderHeader() string {
	if m.player == nil {
		return titleStyle.Render("|-------------------------------------------|")
	}

	current := m.player.GetCurrentTrack()
	songInfo := fmt.Sprintf("Playing: %s - %s", current.Title, current.Artist)

	status := ""
	if m.player.IsPlaying() {
		status = "▶ "
	} else {
		status = "⏸ "
	}

	volumeStr := fmt.Sprintf("Vol: %d%%", int(m.player.GetVolume()*100))
	timeStr := fmt.Sprintf("Time: %s / %s",
		formatTime(m.player.GetCurrentTime()),
		formatTime(m.player.GetTotalTime()))

	left := titleStyle.Render(songInfo)
	center := titleStyle.Render(" ")
	right := titleStyle.Render(fmt.Sprintf("%s | %s | %s |", status, volumeStr, timeStr))

	// Distribute space
	leftWidth := lipgloss.Width(left)
	centerWidth := lipgloss.Width(center)
	rightWidth := lipgloss.Width(right)

	totalUsed := leftWidth + centerWidth + rightWidth
	spacing := (m.width - totalUsed) / 2

	header := left + strings.Repeat(" ", spacing) + center + strings.Repeat(" ", spacing) + right

	return titleStyle.Width(m.width).Render(header)
}

func (m Model) renderProgressBar() string {
	if m.player == nil {
		return ""
	}

	currentTime := m.player.GetCurrentTime()
	totalTime := m.player.GetTotalTime()

	var progressPercent float64
	if totalTime > 0 {
		progressPercent = float64(currentTime) / float64(totalTime)
		if progressPercent > 1 {
			progressPercent = 1
		}
	}

	bar := m.progressBar.ViewAs(progressPercent)
	containerStyle := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Background(colorBg)

	return containerStyle.Render(bar)
}

func (m Model) renderPlaylistColumn() string {
	var b strings.Builder

	title := "--- [ PLAYLISTS ] ---"
	b.WriteString(sectionTitleStyle.Width(m.width/3 - 4).Render(title))
	b.WriteString("\n\n")

	playlists := m.playlistStore.ListPlaylists()

	if len(playlists) == 0 {
		b.WriteString(subtleStyle.Render("No playlists yet"))
		b.WriteString("\n\n")
		b.WriteString(subtleStyle.Render("Press 'c' to create"))
		return b.String()
	}

	maxVisible := (m.height - 16) / 2
	start := 0
	if m.playlistIndex >= maxVisible {
		start = m.playlistIndex - maxVisible + 1
	}

	end := start + maxVisible
	if end > len(playlists) {
		end = len(playlists)
	}

	for i := start; i < end; i++ {
		name := playlists[i]
		trackCount := 0

		if playlist, err := m.playlistStore.GetPlaylist(name); err == nil {
			trackCount = len(playlist.Tracks)
		}

		line := fmt.Sprintf("%s (%d tracks)", name, trackCount)

		if i == m.playlistIndex && m.focusedColumn == 0 {
			b.WriteString(selectedStyle.Render("> " + line))
		} else if name == m.currentPlaylist {
			b.WriteString(statusStyle.Render("* " + line))
		} else {
			b.WriteString(subtleStyle.Render("  " + line))
		}

		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderTracksColumn() string {
	var b strings.Builder

	title := fmt.Sprintf("--- [ TRACKS in '%s' ] ---", m.currentPlaylist)
	if m.currentPlaylist == "" {
		title = "--- [ ALL TRACKS ] ---"
	}

	b.WriteString(sectionTitleStyle.Width(m.width/3 - 4).Render(title))
	b.WriteString("\n\n")

	var tracks []utils.Track
	if m.currentPlaylist != "" {
		if playlist, err := m.playlistStore.GetPlaylist(m.currentPlaylist); err == nil {
			tracks = playlist.Tracks
		}
	} else {
		tracks = m.tracks
	}

	if len(tracks) == 0 {
		b.WriteString(subtleStyle.Render("No tracks"))
		return b.String()
	}

	maxVisible := m.height - 16
	start := 0
	if m.selectedIndex >= maxVisible {
		start = m.selectedIndex - maxVisible + 1
	}

	end := start + maxVisible
	if end > len(tracks) {
		end = len(tracks)
	}

	for i := start; i < end; i++ {
		track := tracks[i]
		line := fmt.Sprintf("%d. %s - %s", i+1, track.Title, track.Artist)

		if i == m.selectedIndex && m.focusedColumn == 1 {
			b.WriteString(selectedStyle.Render("> " + line))
		} else if m.player != nil && track.Path == m.player.GetCurrentTrack().Path {
			b.WriteString(playingStyle.Render("♫ " + line))
		} else {
			b.WriteString(subtleStyle.Render("  " + line))
		}

		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderVisualizerColumn() string {
	var b strings.Builder

	colWidth := (m.width / 3) - 2

	title := "--- [ NOW PLAYING ] ---"
	b.WriteString(sectionTitleStyle.Width(colWidth).Render(title))
	b.WriteString("\n")

	availableHeight := m.height - 14
	infoHeight := 3

	artHeight := availableHeight - infoHeight - 2
	albumArt := m.getAlbumArtBraille(artHeight)

	if albumArt != "" {
		artLines := strings.Split(albumArt, "\n")

		artWidth := 0
		for _, line := range artLines {
			if line != "" {
				lineW := lipgloss.Width(line)
				if lineW > artWidth {
					artWidth = lineW
				}
			}
		}

		for _, line := range artLines {
			if line != "" {
				lineWidth := lipgloss.Width(line)
				padding := (colWidth - lineWidth) / 2
				if padding < 0 {
					padding = 0
				}

				paddedLine := strings.Repeat(" ", padding) + line

				styled := lipgloss.NewStyle().
					Width(colWidth).
					Foreground(colorFg).
					Background(colorBg).
					Render(paddedLine)
				b.WriteString(styled)
				b.WriteString("\n")
			}
		}
	} else {
		b.WriteString(m.renderFallbackVisualizer(artHeight))
	}

	artLinesCount := len(strings.Split(albumArt, "\n"))
	remainingSpace := availableHeight - artLinesCount - infoHeight

	emptyLine := lipgloss.NewStyle().
		Width(colWidth).
		Background(colorBg).
		Render("")

	for i := 0; i < remainingSpace; i++ {
		b.WriteString(emptyLine)
		b.WriteString("\n")
	}

	if m.player != nil {
		current := m.player.GetCurrentTrack()

		infoStyle := lipgloss.NewStyle().
			Foreground(colorSubtle).
			Background(colorBg).
			Width(colWidth).
			Align(lipgloss.Left)

		yearStr := "Unknown"
		if current.Year != 0 {
			yearStr = fmt.Sprintf("%d", current.Year)
		}

		artistLine := infoStyle.Render(fmt.Sprintf("Artist: %s", current.Artist))
		albumLine := infoStyle.Render(fmt.Sprintf("Album: %s", current.Album))
		yearLine := infoStyle.Render(fmt.Sprintf("Year: %s", yearStr))

		b.WriteString(artistLine + "\n")
		b.WriteString(albumLine + "\n")
		b.WriteString(yearLine)
	} else {
		b.WriteString(emptyLine + "\n")
		b.WriteString(emptyLine + "\n")
		b.WriteString(emptyLine)
	}

	return b.String()
}

func (m Model) generateVisualizer() string {
	if m.player == nil || !m.player.IsPlaying() {
		return `
   .  |..  |..|||   ||||   
 ...|...||...|||....|||....
   .  |..  |..|||   ||||   
 ...|...||...|||....|||....
   .  |..  |..|||   ||||   `
	}

	progress := m.player.GetProgress()
	frame := int(progress*10) % 5

	frames := []string{
		`
   |   ||   |||    ||||    
 ...|...||...|||....|||....
   .  |..  |..|||   ||||   
 ...|...||...|||....|||....
   .  |..  |..|||   ||||   `,
		`
   .  |..  |..|||   ||||   
   |   ||   |||    ||||    
 ...|...||...|||....|||....
   .  |..  |..|||   ||||   
 ...|...||...|||....|||....`,
		`
 ...|...||...|||....|||....
   .  |..  |..|||   ||||   
   |   ||   |||    ||||    
 ...|...||...|||....|||....
   .  |..  |..|||   ||||   `,
		`
   .  |..  |..|||   ||||   
 ...|...||...|||....|||....
   |   ||   |||    ||||    
 ...|...||...|||....|||....
   .  |..  |..|||   ||||   `,
		`
 ...|...||...|||....|||....
   .  |..  |..|||   ||||   
 ...|...||...|||....|||....
   |   ||   |||    ||||    
   .  |..  |..|||   ||||   `,
	}

	if frame < len(frames) {
		return frames[frame]
	}
	return frames[0]
}

func (m Model) renderCommands() string {
	commands := "COMMANDS: [C]reate, [D]elete, [ENTER] Select   [A]dd Song, [X]Remove, [SPACE] Play/Pause, [N]ext, [P]rev, [TAB] Switch Column"

	cmdStyle := lipgloss.NewStyle().
		Foreground(colorSubtle).
		Background(colorBg).
		Width(m.width)

	return cmdStyle.Render(commands)
}

func (m Model) renderInput() string {
	var b strings.Builder

	prompt := "Create Playlist"
	if m.inputMode == InputPlaylistLoad {
		prompt = "Load Playlist"
	}

	b.WriteString(headerStyle.Render(prompt) + "\n\n")
	b.WriteString(inputStyle.Render(m.textInput.View()) + "\n\n")
	b.WriteString(subtleStyle.Render("Press ESC to cancel, Enter to confirm"))

	return b.String()
}

func (m Model) renderScanMode() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("📁 Select Music Directory") + "\n\n")
	b.WriteString(inputStyle.Render(m.textInput.View()) + "\n\n")

	if m.scanning {
		b.WriteString(statusStyle.Render("⏳ Scanning directory...") + "\n")
	} else {
		b.WriteString(subtleStyle.Render("Press Enter to start scanning or 'tab' to use file explorer"))
	}

	return b.String()
}

func (m Model) renderExplorerMode() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("📁 File Explorer") + "\n\n")
	b.WriteString(subtleStyle.Render("Current: ") + inputStyle.Render(m.fileExplorer.GetCurrentPath()) + "\n\n")

	if m.fileExplorer.Error != nil {
		b.WriteString(errorStyle.Render("✗ "+m.fileExplorer.Error.Error()) + "\n")
		return b.String()
	}

	maxVisible := m.height - 10
	start := 0
	if m.explorerIndex >= maxVisible {
		start = m.explorerIndex - maxVisible + 1
	}

	end := start + maxVisible
	if end > len(m.fileExplorer.Entries) {
		end = len(m.fileExplorer.Entries)
	}

	for i := start; i < end; i++ {
		entry := m.fileExplorer.Entries[i]
		icon := "📁"
		if entry.Name == ".." {
			icon = "⬆️"
		}

		entryStr := fmt.Sprintf("%s %s", icon, entry.Name)

		if i == m.explorerIndex {
			entryStr = selectedStyle.Render("> " + entryStr)
		} else {
			entryStr = "  " + entryStr
		}

		b.WriteString(entryStr + "\n")
	}

	b.WriteString("\n" + subtleStyle.Render("Space: Enter • Enter: Select • Backspace: Up • ESC: Manual"))

	return b.String()
}

func formatTime(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func formatProgressBar(current, total time.Duration, width int) string {
	if total == 0 {
		emptyBar := lipgloss.NewStyle().
			Foreground(colorSubtle).
			Background(colorBg).
			Render(strings.Repeat("─", width))
		return emptyBar
	}

	ratio := float64(current) / float64(total)
	if ratio > 1 {
		ratio = 1
	}

	filled := int(ratio * float64(width))
	empty := width - filled

	filledBar := lipgloss.NewStyle().
		Foreground(colorAccent).
		Background(colorAccent).
		Render(strings.Repeat("▂", filled))

	emptyBar := lipgloss.NewStyle().
		Foreground(colorSubtle).
		Background(colorSubtle).
		Render(strings.Repeat("▂", empty))

	return filledBar + emptyBar
}

func scanTracks(dir string) tea.Cmd {
	return func() tea.Msg {
		tracks, err := utils.ScanDir(dir)
		return scanMsg{tracks: tracks, err: err}
	}
}

func (m Model) getAlbumArtBraille(height int) string {
	if m.player == nil {
		return ""
	}

	currentTrack := m.player.GetCurrentTrack()

	if currentTrack.Path == m.cachedTrackPath && m.cachedAlbumArt != "" {
		return m.cachedAlbumArt
	}

	colWidth := (m.width / 3) - 2
	squareWidth := height * 2

	width := colWidth
	if squareWidth < colWidth {
		width = squareWidth
	}

	art := utils.GetAlbumArtBrailleColored(currentTrack.Path, width, height)

	if art == "" {
		return ""
	}

	m.cachedAlbumArt = art
	m.cachedTrackPath = currentTrack.Path

	return art
}

func (m Model) renderFallbackVisualizer(height int) string {
	var b strings.Builder

	visualizerHeight := 7
	topPadding := (height - visualizerHeight) / 2

	emptyLine := lipgloss.NewStyle().
		Width(m.width/3 - 4).
		Background(colorBg).
		Render("")

	for i := 0; i < topPadding; i++ {
		b.WriteString(emptyLine)
		b.WriteString("\n")
	}

	visualizer := m.generateVisualizer()
	visualizerLines := strings.Split(visualizer, "\n")
	for _, line := range visualizerLines {
		if line != "" {
			centered := lipgloss.NewStyle().
				Width(m.width/3 - 4).
				Align(lipgloss.Center).
				Foreground(colorAccent).
				Background(colorBg).
				Render(line)
			b.WriteString(centered)
			b.WriteString("\n")
		}
	}

	return b.String()
}
