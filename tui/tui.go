package tui

import (
    "fmt"
    "strings"
    "time"

    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/bubbles/viewport"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/ryansantos40/go-music-player/utils"
)

type Model struct {
    viewport       viewport.Model
    textInput      textinput.Model
    tracks         []utils.Track
    scanning       bool
    player         *utils.Player
    selectedIndex  int
    mode           AppMode
    lastTrackIdx   int
    playlistStore  *utils.PlaylistStore
    currentPlaylist string
    inputMode      InputMode
    errorMsg       string
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
    ti.Placeholder = "Enter directory to scan"
    ti.Focus()

    vp := viewport.New(80, 20)

    playlistStore, _ := utils.NewPlaylistStore()

    return Model{
        viewport:       vp,
        textInput:      ti,
        selectedIndex:  0,
        mode:           ModeScan,
        lastTrackIdx:   -1,
        playlistStore:  playlistStore,
        currentPlaylist: "",
        inputMode:      InputNone,
        errorMsg:       "",
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
        switch msg.String() {
        case "ctrl+c", "q":
            if m.player != nil {
                m.player.Stop()
            }
            return m, tea.Quit

        case "esc":
            m.inputMode = InputNone
            m.textInput.Reset()
            m.errorMsg = ""

        case "enter":
            if m.inputMode == InputPlaylistName {
                name := m.textInput.Value()
                if err := m.playlistStore.CreatePlaylist(name); err != nil {
                    m.errorMsg = err.Error()
                } else {
                    m.currentPlaylist = name
                    m.inputMode = InputNone
                }
                m.textInput.Reset()

            } else if m.inputMode == InputPlaylistLoad {
                name := m.textInput.Value()
                if _, err := m.playlistStore.GetPlaylist(name); err != nil {
                    m.errorMsg = err.Error()
                } else {
                    m.currentPlaylist = name
                    m.mode = ModePlaylist
                    m.inputMode = InputNone
                }
                m.textInput.Reset()

            } else if m.mode == ModeScan && !m.scanning && m.inputMode == InputNone {
                m.scanning = true
                return m, scanTracks(m.textInput.Value())

            } else if m.mode == ModePlayer && m.player != nil {
                m.player.Skip(m.selectedIndex)

            } else if m.mode == ModePlaylist && m.currentPlaylist != "" {
                playlist, _ := m.playlistStore.GetPlaylist(m.currentPlaylist)
                if m.selectedIndex < len(playlist.Tracks) {
                    m.player = utils.NewPlayer(playlist.Tracks)
                    m.player.Skip(m.selectedIndex)
                    m.mode = ModePlayer
                }
            }

        case " ":
            if m.mode == ModePlayer && m.player != nil {
                if m.player.IsPlaying() {
                    m.player.Pause()
                } else {
                    m.player.Resume()
                }
            }

        case "n":
            if m.mode == ModePlayer && m.player != nil {
                m.player.Next()
            }

        case "p":
            if m.mode == ModePlayer && m.player != nil {
                m.player.Previous()
            }

        case "s":
            if m.mode == ModePlayer && m.player != nil {
                m.player.ToggleShuffle()
            }

        case "r":
            if m.mode == ModePlayer && m.player != nil {
                m.player.ToggleRepeat()
            }

        case "up":
            if m.inputMode == InputNone {
                if m.mode == ModePlayer || m.mode == ModePlaylist {
                    m.selectedIndex = (m.selectedIndex - 1 + len(m.tracks)) % len(m.tracks)
                }
            }

        case "down":
            if m.inputMode == InputNone {
                if m.mode == ModePlayer || m.mode == ModePlaylist {
                    m.selectedIndex = (m.selectedIndex + 1) % len(m.tracks)
                }
            }

        case "c":
            if m.inputMode == InputNone && m.mode == ModePlayer {
                m.inputMode = InputPlaylistName
                m.textInput.Focus()
                m.textInput.Placeholder = "Playlist name"

            }

        case "l":
            if m.inputMode == InputNone && m.mode == ModePlayer {
                m.inputMode = InputPlaylistLoad
                m.textInput.Focus()
                m.textInput.Placeholder = "Playlist name to load"

            }

        case "a":
            if m.inputMode == InputNone && m.mode == ModePlaylist && m.player != nil && m.currentPlaylist != "" {
                track := m.player.GetCurrentTrack()
                if err := m.playlistStore.AddTrack(m.currentPlaylist, track); err != nil {
                    m.errorMsg = err.Error()
                }
            }

        case "d":
            if m.inputMode == InputNone && m.mode == ModePlaylist && m.currentPlaylist != "" {
                if err := m.playlistStore.RemoveTrack(m.currentPlaylist, m.selectedIndex); err != nil {
                    m.errorMsg = err.Error()
                }
            }

        case "tab":
            if m.mode == ModeScan && len(m.tracks) > 0 {
                m.mode = ModePlayer
                m.player = utils.NewPlayer(m.tracks)
                m.player.Play()
                m.lastTrackIdx = m.player.GetCurrentIndex()
            }
        }

    case scanMsg:
        m.scanning = false
        if msg.err != nil {
            m.errorMsg = "Error scanning directory: " + msg.err.Error()
        } else {
            m.tracks = msg.tracks
            m.mode = ModePlayer
            m.player = utils.NewPlayer(m.tracks)
            m.player.Play()
            m.lastTrackIdx = m.player.GetCurrentIndex()
        }

    case tea.WindowSizeMsg:
        m.viewport.Width = msg.Width
        m.viewport.Height = msg.Height - 8

    case tickMsg:
        if m.mode == ModePlayer && m.player != nil && m.player.IsPlaying() {
            currentTime := m.player.GetCurrentTime()
            totalTime := m.player.GetTotalTime()
            currentIdx := m.player.GetCurrentIndex()

            if currentIdx != m.lastTrackIdx {
                m.lastTrackIdx = currentIdx
            } else if totalTime > 0 && currentTime >= totalTime {
                m.player.HandleTrackEnd()
                m.lastTrackIdx = m.player.GetCurrentIndex()
            }
        }
        cmd = tick()
    }

    m.textInput, cmd = m.textInput.Update(msg)
    m.viewport, _ = m.viewport.Update(msg)
    return m, cmd
}

func (m Model) View() string {
    var b strings.Builder

    b.WriteString("🎵 Music Player TUI\n")
    b.WriteString("═══════════════════════════════════════════════════════════════════════════════\n")

    if m.errorMsg != "" {
        b.WriteString(fmt.Sprintf("❌ Error: %s\n\n", m.errorMsg))
    }

    if m.inputMode != InputNone {
        b.WriteString(m.textInput.View())
        b.WriteString("\n")
    }

    if m.mode == ModeScan {
        b.WriteString("📁 Scan Mode\n")
        b.WriteString("Type the directory and press <enter>\n")
        b.WriteString(m.textInput.View())
        b.WriteString("\n\n")

        if m.scanning {
            b.WriteString("⏳ Scanning...\n")
        }

        b.WriteString(m.viewport.View())

    } else if m.mode == ModePlayer {
        if m.player != nil {
            current := m.player.GetCurrentTrack()
            b.WriteString(fmt.Sprintf("Now Playing: %s - %s\n", current.Title, current.Artist))
            b.WriteString(fmt.Sprintf("Album: %s\n", current.Album))

            status := "⏸ Paused"
            if m.player.IsPlaying() {
                status = "▶ Playing"
            }
            shuffle := "Shuffle: Off"
            if m.player.GetShuffle() {
                shuffle = "Shuffle: On"
            }
            repeat := fmt.Sprintf("Repeat: %s", m.player.GetRepeatMode().String())

            b.WriteString(fmt.Sprintf("%s | %s | %s\n", status, shuffle, repeat))

            current_time := m.player.GetCurrentTime()
            total_time := m.player.GetTotalTime()
            progress := formatProgressBar(current_time, total_time, 60)
            b.WriteString(fmt.Sprintf("Progress: %s %s / %s\n", progress, formatTime(current_time), formatTime(total_time)))

            b.WriteString("\n─────────────────────────────────────────────────────────────────────────────\n")
            b.WriteString("Tracks in Library:\n")
            b.WriteString(m.renderTrackList())
        }

    } else if m.mode == ModePlaylist {
        b.WriteString(fmt.Sprintf("📋 Playlist: %s\n", m.currentPlaylist))
        if playlist, err := m.playlistStore.GetPlaylist(m.currentPlaylist); err == nil {
            b.WriteString(fmt.Sprintf("Tracks: %d\n\n", len(playlist.Tracks)))
            for i, track := range playlist.Tracks {
                prefix := "  "
                if i == m.selectedIndex {
                    prefix = "➤ "
                }
                b.WriteString(fmt.Sprintf("%s[%d] %s - %s\n", prefix, i+1, track.Title, track.Artist))
            }
        }
    }

    b.WriteString("\n─────────────────────────────────────────────────────────────────────────────\n")
    controls := "Controls: [space] play/pause | [n] next | [p] prev | [s] shuffle | [r] repeat | [↑↓] select | [enter] play | [c] create playlist | [l] load playlist | [a] add track | [d] delete track | [q] quit\n"
    b.WriteString(controls)

    return b.String()
}

func (m Model) renderTrackList() string {
    var b strings.Builder

    for i, track := range m.tracks {
        prefix := "  "
        if i == m.selectedIndex {
            prefix = "➤ "
        }

        if i == m.player.GetCurrentIndex() {
            prefix = "▶ "
        }

        b.WriteString(fmt.Sprintf("%s[%d] %s - %s\n", prefix, i+1, track.Title, track.Artist))
    }

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
        return "[" + strings.Repeat("─", width) + "]"
    }

    ratio := float64(current) / float64(total)
    if ratio > 1 {
        ratio = 1
    }

    filled := int(ratio * float64(width))
    bar := "[" + strings.Repeat("█", filled) + strings.Repeat("─", width-filled) + "]"
    return bar
}

func scanTracks(dir string) tea.Cmd {
    return func() tea.Msg {
        tracks, err := utils.ScanDir(dir)
        return scanMsg{tracks: tracks, err: err}
    }
}