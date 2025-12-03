package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderHeader() string {
	if m.player == nil {
		return titleStyle.Render("|-------------------------------------------|")
	}

	current := m.player.GetCurrentTrack()
	songInfo := fmt.Sprintf("Playing: %s - %s", current.Title, current.Artist)

	status := "⏸ "
	if m.player.IsPlaying() {
		status = "▶ "
	}

	volumeStr := fmt.Sprintf("Vol: %d%%", int(m.player.GetVolume()*100))
	timeStr := fmt.Sprintf("Time: %s / %s",
		formatTime(m.player.GetCurrentTime()),
		formatTime(m.player.GetTotalTime()))

	left := titleStyle.Render(songInfo)
	center := titleStyle.Render(" ")
	right := titleStyle.Render(fmt.Sprintf("%s | %s | %s |", status, volumeStr, timeStr))

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
