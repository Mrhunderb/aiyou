package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Mrhunderb/ayou/internal/pactl"
	"github.com/Mrhunderb/ayou/internal/ui"
)

func main() {
	if _, err := exec.LookPath("pactl"); err != nil {
		fmt.Fprintln(os.Stderr, "ayou requires pactl (PulseAudio or pipewire-pulse)")
		os.Exit(1)
	}
	p := tea.NewProgram(ui.New(pactl.New()), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
