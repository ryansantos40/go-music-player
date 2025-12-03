package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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

	b.WriteString(m.renderHeader())
	b.WriteString("\n")
	b.WriteString(m.renderProgressBar())
	b.WriteString("\n")
	b.WriteString(m.renderColumns())
	b.WriteString("\n")
	b.WriteString(m.renderCommands())

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

func (m Model) renderCommands() string {
	commands := "COMMANDS: [C]reate, [D]elete, [ENTER] Select   [A]dd Song, [X]Remove, [SPACE] Play/Pause, [N]ext, [P]rev, [TAB] Switch Column"

	cmdStyle := lipgloss.NewStyle().
		Foreground(colorSubtle).
		Background(colorBg).
		Width(m.width)

	return cmdStyle.Render(commands)
}
