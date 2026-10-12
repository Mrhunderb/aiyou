package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Mrhunderb/ayou/internal/config"
	"github.com/Mrhunderb/ayou/internal/pactl"
	"github.com/Mrhunderb/ayou/internal/ui"
)

func main() {
	configPath := flag.String("config", "", "configuration JSON file (default: user config directory)")
	printDefaults := flag.Bool("print-default-config", false, "print default configuration as JSON and exit")
	flag.Parse()
	if *printDefaults {
		out, err := json.MarshalIndent(config.Default(), "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(string(out))
		return
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := exec.LookPath("pactl"); err != nil {
		fmt.Fprintln(os.Stderr, "ayou requires pactl (PulseAudio or pipewire-pulse)")
		os.Exit(1)
	}
	backend := pactl.New(cfg.Backend.PactlTimeout.Duration(), cfg.Backend.BluetoothTimeout.Duration())
	p := tea.NewProgram(ui.New(backend, cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
