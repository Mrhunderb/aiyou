package ui

import (
	"time"

	"github.com/Mrhunderb/ayou/internal/audio"
	"github.com/Mrhunderb/ayou/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	outputs = iota
	inputs
	playback
	recording
	configuration
	panelCount
)

var titles = []string{"OUTPUT DEVICES", "INPUT DEVICES", "PLAYBACK", "RECORDING", "CONFIGURATION"}

type refreshMsg struct {
	data audio.Snapshot
	err  error
}
type actionMsg struct {
	text string
	err  error
}
type tickMsg struct{}

type menuOption struct {
	label  string
	action audio.Action
}
type model struct {
	backend    audio.Backend
	config     config.Config
	theme      theme
	data       audio.Snapshot
	focus      int
	cursor     [panelCount]int
	scroll     [panelCount]int
	width      int
	height     int
	status     string
	expanded   bool
	help       bool
	searching  bool
	search     string
	menuTitle  string
	menu       []menuOption
	menuCursor int
}

// New wires validated configuration and the backend to the Bubble Tea model.
func New(backend audio.Backend, cfg config.Config) tea.Model {
	return model{backend: backend, config: cfg, theme: newTheme(cfg.Theme)}
}

func (m model) refresh() tea.Cmd {
	return func() tea.Msg { s, err := m.backend.Snapshot(); return refreshMsg{s, err} }
}
func (m model) runAction(action audio.Action) tea.Cmd {
	return func() tea.Msg { err := m.backend.Execute(action); return actionMsg{action.String(), err} }
}
func (m model) tick() tea.Cmd {
	return tea.Tick(m.config.RefreshInterval.Duration(), func(time.Time) tea.Msg { return tickMsg{} })
}
func (m model) Init() tea.Cmd { return tea.Batch(m.refresh(), m.tick()) }
