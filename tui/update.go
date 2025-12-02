package tui

import (
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ryansantos40/go-music-player/utils"
)

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

		if m.mode == ModePlayer {
			switch msg.String() {
			case "tab":
				m.focusedColumn = (m.focusedColumn + 1) % 2
			case "up", "k":
				if m.focusedColumn == 0 {
					playlists := m.playlistStore.ListPlaylists()
					if len(playlists) > 0 {
						m.playlistIndex = (m.playlistIndex - 1 + len(playlists)) % len(playlists)
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
