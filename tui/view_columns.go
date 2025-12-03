package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderColumns() string {
	var b strings.Builder

	col1Width := m.width / 3
	col2Width := m.width / 3
	col3Width := m.width - col1Width - col2Width

	col1ContentWidth := col1Width - 2
	col2ContentWidth := col2Width - 2
	col3ContentWidth := col3Width - 2

	colHeight := m.height - 14

	col1Lines := m.getColumnLines(m.renderLibraryColumn(), colHeight)
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
