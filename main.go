package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ryansantos40/go-music-player/tui"
)

func main() {
	m := tui.NewModel()
	p := tea.NewProgram(m)
	_, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
