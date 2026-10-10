package ui

import (
	"time"

	"github.com/Mrhunderb/ayou/internal/audio"
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

// New wires the backend to the Bubble Tea model.
func New(backend audio.Backend) tea.Model { return model{backend: backend} }

func (m model) refresh() tea.Cmd {
	return func() tea.Msg { s, err := m.backend.Snapshot(); return refreshMsg{s, err} }
}
func (m model) runAction(action audio.Action) tea.Cmd {
	return func() tea.Msg { err := m.backend.Execute(action); return actionMsg{action.String(), err} }
}
func tick() tea.Cmd           { return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return tickMsg{} }) }
func (m model) Init() tea.Cmd { return tea.Batch(m.refresh(), tick()) }
