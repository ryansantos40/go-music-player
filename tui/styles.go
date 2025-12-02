package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorBg      = lipgloss.Color("")
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
