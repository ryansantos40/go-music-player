package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ryansantos40/go-music-player/utils"
)

func (m Model) renderVisualizerColumn() string {
	var b strings.Builder

	colWidth := (m.width / 3)

	title := "--- [ NOW PLAYING ] ---"
	b.WriteString(sectionTitleStyle.Width(colWidth).Render(title))
	b.WriteString("\n")

	availableHeight := m.height - 14
	infoHeight := 3

	artHeight := availableHeight - infoHeight - 1
	albumArt := m.getAlbumArtBraille(artHeight)

	if albumArt != "" {
		artLines := strings.Split(albumArt, "\n")

		for _, line := range artLines {
			if line == "" {
				continue
			}
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

	art := utils.GetAlbumArtHalfBlocksColored(currentTrack.Path, width, height)

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
