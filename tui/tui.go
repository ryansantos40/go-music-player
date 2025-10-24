package tui

import (
    "fmt"
    "strings"
    "time"

    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/bubbles/viewport"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/ryansantos40/go-music-player/utils"
)

// Estilos usando lipgloss
var (
    titleStyle = lipgloss.NewStyle().
            Bold(true).
            Foreground(lipgloss.Color("#FF06B7")).
            MarginBottom(1)

    headerStyle = lipgloss.NewStyle().
            Bold(true).
            Foreground(lipgloss.Color("#7D56F4"))

    statusStyle = lipgloss.NewStyle().
            Foreground(lipgloss.Color("#04B575"))

    errorStyle = lipgloss.NewStyle().
            Foreground(lipgloss.Color("#FF0000")).
            Bold(true)

    selectedStyle = lipgloss.NewStyle().
            Foreground(lipgloss.Color("#FFF")).
            Background(lipgloss.Color("#7D56F4")).
            Bold(true)

    playingStyle = lipgloss.NewStyle().
            Foreground(lipgloss.Color("#04B575")).
            Bold(true)

    subtleStyle = lipgloss.NewStyle().
            Foreground(lipgloss.Color("#888"))

    helpStyle = lipgloss.NewStyle().
            Foreground(lipgloss.Color("#666")).
            MarginTop(1)
)

type Model struct {
    viewport        viewport.Model
    textInput       textinput.Model
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
}

type AppMode int
type InputMode int

const (
    ModeScan AppMode = iota
    ModePlayer
    ModePlaylist
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

    vp := viewport.New(80, 20)

    playlistStore, _ := utils.NewPlaylistStore()

    return Model{
        viewport:        vp,
        textInput:       ti,
        selectedIndex:   0,
        mode:            ModeScan,
        lastTrackIdx:    -1,
        playlistStore:   playlistStore,
        currentPlaylist: "",
        inputMode:       InputNone,
        errorMsg:        "",
        width:           80,
        height:          24,
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

    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "ctrl+c" || (msg.String() == "q" && m.inputMode == InputNone) {
            if m.player != nil {
                m.player.Stop()
            }
            return m, tea.Quit
        }

        if msg.String() == "esc" {
            m.inputMode = InputNone
            m.textInput.Reset()
            m.errorMsg = ""
            return m, nil
        }

        if m.inputMode != InputNone {
            if msg.String() == "enter" {
                m = m.handleInputSubmit()
                return m, nil
            }
            m.textInput, cmd = m.textInput.Update(msg)
            return m, cmd
        }

        switch msg.String() {
        case "up", "k":
            if m.mode == ModePlayer || m.mode == ModePlaylist {
                tracksLen := len(m.tracks)
                if m.mode == ModePlaylist && m.currentPlaylist != "" {
                    if playlist, err := m.playlistStore.GetPlaylist(m.currentPlaylist); err == nil {
                        tracksLen = len(playlist.Tracks)
                    }
                }
                if tracksLen > 0 {
                    m.selectedIndex = (m.selectedIndex - 1 + tracksLen) % tracksLen
                }
            }

        case "down", "j":
            if m.mode == ModePlayer || m.mode == ModePlaylist {
                tracksLen := len(m.tracks)
                if m.mode == ModePlaylist && m.currentPlaylist != "" {
                    if playlist, err := m.playlistStore.GetPlaylist(m.currentPlaylist); err == nil {
                        tracksLen = len(playlist.Tracks)
                    }
                }
                if tracksLen > 0 {
                    m.selectedIndex = (m.selectedIndex + 1) % tracksLen
                }
            }

        case "enter":
            m = m.handleEnter()
            return m, scanTracks(m.textInput.Value())

        case " ":
            if m.mode == ModePlayer && m.player != nil {
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
            if m.mode == ModePlayer {
                m.inputMode = InputPlaylistName
                m.textInput.Placeholder = "Enter playlist name..."
                m.textInput.Focus()
            }

        case "L":
            if m.mode == ModePlayer {
                m.inputMode = InputPlaylistLoad
                m.textInput.Placeholder = "Enter playlist name to load..."
                m.textInput.Focus()
            }

        case "a":
            if m.mode == ModePlaylist && m.player != nil && m.currentPlaylist != "" {
                track := m.player.GetCurrentTrack()
                if err := m.playlistStore.AddTrack(m.currentPlaylist, track); err != nil {
                    m.errorMsg = err.Error()
                }
            }

        case "d":
            if m.mode == ModePlaylist && m.currentPlaylist != "" {
                if err := m.playlistStore.RemoveTrack(m.currentPlaylist, m.selectedIndex); err != nil {
                    m.errorMsg = err.Error()
                }
            }

        case "1":
            if m.mode != ModeScan {
                m.mode = ModePlayer
                m.errorMsg = ""
            }

        case "2":
            m.mode = ModePlaylist
            m.errorMsg = ""

        case "?":
            // Placeholder
        }

    case scanMsg:
        m.scanning = false
        if msg.err != nil {
            m.errorMsg = "Error: " + msg.err.Error()
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

    case tickMsg:
        if m.mode == ModePlayer && m.player != nil && m.player.IsPlaying() {
            currentIdx := m.player.GetCurrentIndex()
            if currentIdx != m.lastTrackIdx {
                m.lastTrackIdx = currentIdx
            }
        }
        cmd = tick()
    }

    return m, cmd
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
            m.mode = ModePlaylist
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
    
    if m.mode == ModePlayer && m.player != nil && len(m.tracks) > 0 {
        m.player.Skip(m.selectedIndex)
        return m
    }
    
    if m.mode == ModePlaylist && m.currentPlaylist != "" {
        if playlist, err := m.playlistStore.GetPlaylist(m.currentPlaylist); err == nil {
            if m.selectedIndex < len(playlist.Tracks) {
                m.player = utils.NewPlayer(playlist.Tracks)
                m.player.Skip(m.selectedIndex)
                m.mode = ModePlayer
            }
        }
    }
    
    return m
}

func (m Model) View() string {
    var b strings.Builder

    // Título
    title := titleStyle.Render("♫ Go Music Player")
    b.WriteString(title + "\n")
    b.WriteString(strings.Repeat("─", m.width) + "\n\n")

    if m.errorMsg != "" {
        b.WriteString(errorStyle.Render("✗ " + m.errorMsg))
        b.WriteString("\n\n")
    }

    if m.inputMode != InputNone {
        b.WriteString(m.renderInput())
        b.WriteString("\n")
        return b.String()
    }

    switch m.mode {
    case ModeScan:
        b.WriteString(m.renderScanMode())
    case ModePlayer:
        b.WriteString(m.renderPlayerMode())
    case ModePlaylist:
        b.WriteString(m.renderPlaylistMode())
    }

    b.WriteString("\n" + strings.Repeat("─", m.width) + "\n")
    b.WriteString(m.renderControls())

    return b.String()
}

func (m Model) renderInput() string {
    var b strings.Builder
    
    prompt := "Create Playlist"
    if m.inputMode == InputPlaylistLoad {
        prompt = "Load Playlist"
    }
    
    b.WriteString(headerStyle.Render(prompt) + "\n")
    b.WriteString(m.textInput.View() + "\n")
    b.WriteString(subtleStyle.Render("Press ESC to cancel, Enter to confirm"))
    
    return b.String()
}

func (m Model) renderScanMode() string {
    var b strings.Builder

    b.WriteString(headerStyle.Render("📁 Select Music Directory") + "\n\n")
    b.WriteString(m.textInput.View() + "\n\n")

    if m.scanning {
        b.WriteString(statusStyle.Render("⏳ Scanning directory...") + "\n")
    } else {
        b.WriteString(subtleStyle.Render("Press Enter to start scanning"))
    }

    return b.String()
}

func (m Model) renderPlayerMode() string {
    var b strings.Builder

    if m.player == nil {
        return subtleStyle.Render("No tracks loaded")
    }

    current := m.player.GetCurrentTrack()
    b.WriteString(headerStyle.Render("♫ Now Playing") + "\n")
    b.WriteString(fmt.Sprintf("%s\n", playingStyle.Render(current.Title)))
    b.WriteString(subtleStyle.Render(fmt.Sprintf("%s • %s", current.Artist, current.Album)) + "\n\n")

    status := m.renderStatusBar()
    b.WriteString(status + "\n\n")

    progress := m.renderProgressBar()
    b.WriteString(progress + "\n\n")

    b.WriteString(headerStyle.Render(fmt.Sprintf("Library (%d tracks)", len(m.tracks))) + "\n")
    b.WriteString(m.renderTrackList())

    return b.String()
}

func (m Model) renderPlaylistMode() string {
    var b strings.Builder

    b.WriteString(headerStyle.Render(fmt.Sprintf("📋 Playlist: %s", m.currentPlaylist)) + "\n")

    if playlist, err := m.playlistStore.GetPlaylist(m.currentPlaylist); err == nil {
        b.WriteString(subtleStyle.Render(fmt.Sprintf("%d tracks", len(playlist.Tracks))) + "\n\n")
        
        for i, track := range playlist.Tracks {
            prefix := "  "
            trackStr := fmt.Sprintf("[%d] %s - %s", i+1, track.Title, track.Artist)
            
            if i == m.selectedIndex {
                trackStr = selectedStyle.Render("▶ " + trackStr)
            } else {
                trackStr = prefix + trackStr
            }
            
            b.WriteString(trackStr + "\n")
        }
    }

    return b.String()
}

func (m Model) renderStatusBar() string {
    if m.player == nil {
        return ""
    }

    var parts []string

    if m.player.IsPlaying() {
        parts = append(parts, statusStyle.Render("▶ Playing"))
    } else {
        parts = append(parts, subtleStyle.Render("⏸ Paused"))
    }

    if m.player.GetShuffle() {
        parts = append(parts, "🔀 Shuffle")
    }

    parts = append(parts, fmt.Sprintf("%s", m.player.GetRepeatMode().String()))

    parts = append(parts, fmt.Sprintf("🔊 %d%%", int(m.player.GetVolume()*100)))

    return strings.Join(parts, " │ ")
}

func (m Model) renderProgressBar() string {
    if m.player == nil {
        return ""
    }

    currentTime := m.player.GetCurrentTime()
    totalTime := m.player.GetTotalTime()

    barWidth := 50
    if m.width > 80 {
        barWidth = m.width - 30
    }

    bar := formatProgressBar(currentTime, totalTime, barWidth)
    timeStr := fmt.Sprintf("%s / %s", formatTime(currentTime), formatTime(totalTime))

    return fmt.Sprintf("%s  %s", bar, subtleStyle.Render(timeStr))
}

func (m Model) renderTrackList() string {
    var b strings.Builder

    maxVisible := 10
    if m.height > 30 {
        maxVisible = m.height - 20
    }

    start := 0
    if m.selectedIndex >= maxVisible {
        start = m.selectedIndex - maxVisible + 1
    }

    end := start + maxVisible
    if end > len(m.tracks) {
        end = len(m.tracks)
    }

    for i := start; i < end; i++ {
        track := m.tracks[i]
        prefix := "  "
        trackStr := fmt.Sprintf("[%3d] %-40s %s", i+1, 
            truncate(track.Title, 40), 
            subtleStyle.Render(track.Artist))

        if i == m.selectedIndex {
            trackStr = selectedStyle.Render("▶ " + trackStr)
        } else if m.player != nil && i == m.player.GetCurrentIndex() {
            trackStr = playingStyle.Render("♫ " + trackStr)
        } else {
            trackStr = prefix + trackStr
        }

        b.WriteString(trackStr + "\n")
    }

    if len(m.tracks) > maxVisible {
        remaining := len(m.tracks) - end
        if remaining > 0 {
            b.WriteString(subtleStyle.Render(fmt.Sprintf("\n... %d more tracks", remaining)))
        }
    }

    return b.String()
}

func (m Model) renderControls() string {
    controls := []string{
        "[1] Library",
        "[2] Playlists",
        "[space] Play/Pause",
        "[n/p] Next/Prev",
        "[←→] Seek",
        "[+/-] Volume",
        "[s] Shuffle",
        "[r] Repeat",
        "[c] Create",
        "[L] Load",
        "[?] Help",
        "[q] Quit",
    }

    return helpStyle.Render(strings.Join(controls, " • "))
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
        return "[" + strings.Repeat("─", width) + "]"
    }

    ratio := float64(current) / float64(total)
    if ratio > 1 {
        ratio = 1
    }

    filled := int(ratio * float64(width))
    empty := width - filled

    return "[" + strings.Repeat("━", filled) + strings.Repeat("─", empty) + "]"
}

func truncate(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen-3] + "..."
}

func scanTracks(dir string) tea.Cmd {
    return func() tea.Msg {
        tracks, err := utils.ScanDir(dir)
        return scanMsg{tracks: tracks, err: err}
    }
}