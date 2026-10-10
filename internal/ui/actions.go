package ui

import (
	"sort"

	"github.com/Mrhunderb/aiyou/internal/audio"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) volumeAction(change string) tea.Cmd {
	id := m.selectedID(m.focus)
	if id == "" || m.focus == configuration {
		return nil
	}
	op := []audio.Operation{audio.SetPlaybackVolume, audio.SetRecordingVolume, audio.SetOutputVolume, audio.SetInputVolume}[m.focus]
	return m.runAction(audio.Action{Operation: op, Target: id, Value: change})
}
func (m model) muteAction() tea.Cmd {
	id := m.selectedID(m.focus)
	if id == "" || m.focus == configuration {
		return nil
	}
	op := []audio.Operation{audio.TogglePlaybackMute, audio.ToggleRecordingMute, audio.ToggleOutputMute, audio.ToggleInputMute}[m.focus]
	return m.runAction(audio.Action{Operation: op, Target: id})
}
func (m model) defaultAction() tea.Cmd {
	id := m.selectedID(m.focus)
	if id == "" {
		return nil
	}
	if m.focus == outputs {
		return m.runAction(audio.Action{Operation: audio.SetDefaultOutput, Target: id})
	}
	if m.focus == inputs {
		return m.runAction(audio.Action{Operation: audio.SetDefaultInput, Target: id})
	}
	return nil
}
func (m *model) openMenu() {
	id := m.selectedID(m.focus)
	if id == "" {
		return
	}
	m.menuCursor = 0
	m.menu = nil
	switch m.focus {
	case playback:
		m.menuTitle = "MOVE PLAYBACK TO OUTPUT"
		for _, s := range m.data.Sinks {
			m.menu = append(m.menu, menuOption{s.DisplayName, audio.Action{Operation: audio.MovePlayback, Target: id, Value: s.Name}})
		}
	case recording:
		m.menuTitle = "MOVE RECORDING TO INPUT"
		for _, s := range m.data.Sources {
			m.menu = append(m.menu, menuOption{s.DisplayName, audio.Action{Operation: audio.MoveRecording, Target: id, Value: s.Name}})
		}
	case outputs:
		m.menuTitle = "OUTPUT PORT"
		for _, p := range m.data.Sinks[m.cursor[outputs]].Ports {
			m.menu = append(m.menu, menuOption{p.Description + " (" + p.Availability + ")", audio.Action{Operation: audio.SetOutputPort, Target: id, Value: p.Name}})
		}
	case inputs:
		m.menuTitle = "INPUT PORT"
		for _, p := range m.data.Sources[m.cursor[inputs]].Ports {
			m.menu = append(m.menu, menuOption{p.Description + " (" + p.Availability + ")", audio.Action{Operation: audio.SetInputPort, Target: id, Value: p.Name}})
		}
	case configuration:
		m.menuTitle = "CARD PROFILE"
		card := m.data.Cards[m.cursor[configuration]]
		keys := make([]string, 0, len(card.Profiles))
		for key := range card.Profiles {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			p := card.Profiles[key]
			mark := "  "
			if key == card.ActiveProfile {
				mark = "* "
			}
			if !p.Available {
				mark += "(unavailable) "
			}
			m.menu = append(m.menu, menuOption{mark + p.Description, audio.Action{Operation: audio.SetCardProfile, Target: id, Value: key}})
		}
	}
	if len(m.menu) == 0 {
		m.status = "No options available"
	}
}
