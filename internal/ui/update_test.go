package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/Mrhunderb/aiyou/internal/audio"
	tea "github.com/charmbracelet/bubbletea"
)

type stubBackend struct {
	snapshot audio.Snapshot
	actions  []audio.Action
	err      error
}

func (s *stubBackend) Snapshot() (audio.Snapshot, error) { return s.snapshot, s.err }
func (s *stubBackend) Execute(a audio.Action) error      { s.actions = append(s.actions, a); return s.err }

func TestRefreshRetainsSelection(t *testing.T) {
	backend := &stubBackend{}
	m := New(backend).(model)
	backend.snapshot = sampleModel().data
	result := m.refresh()().(refreshMsg)
	updated, _ := m.Update(result)
	m = updated.(model)
	m.focus = outputs
	m.cursor[outputs] = 1
	backend.snapshot.Sinks = append([]audio.Sink(nil), backend.snapshot.Sinks...)
	backend.snapshot.Sinks[0], backend.snapshot.Sinks[1] = backend.snapshot.Sinks[1], backend.snapshot.Sinks[0]
	result = m.refresh()().(refreshMsg)
	updated, _ = m.Update(result)
	m = updated.(model)
	if m.cursor[outputs] != 0 || m.selectedID(outputs) != "headphones" {
		t.Fatalf("selection was not preserved: cursor=%d id=%s", m.cursor[outputs], m.selectedID(outputs))
	}
	backend.err = errors.New("server unavailable")
	updated, _ = m.Update(m.refresh()())
	m = updated.(model)
	if m.selectedID(outputs) != "headphones" || !strings.Contains(m.status, "server unavailable") {
		t.Fatalf("error should retain previous snapshot: %+v", m)
	}
}
func TestNumberedPanelActions(t *testing.T) {
	backend := &stubBackend{}
	for _, tc := range []struct {
		key          string
		focus        int
		id           string
		volume, mute audio.Operation
	}{
		{"1", outputs, "speakers", audio.SetOutputVolume, audio.ToggleOutputMute},
		{"2", inputs, "microphone", audio.SetInputVolume, audio.ToggleInputMute},
		{"3", playback, "4", audio.SetPlaybackVolume, audio.TogglePlaybackMute},
		{"4", recording, "5", audio.SetRecordingVolume, audio.ToggleRecordingMute},
	} {
		m := sampleModel()
		m.backend = backend
		m = key(m, tc.key)
		if m.focus != tc.focus || m.selectedID(m.focus) != tc.id {
			t.Fatalf("key %s focused %d / %q", tc.key, m.focus, m.selectedID(m.focus))
		}
		for _, op := range []struct {
			key       string
			operation audio.Operation
		}{{"+", tc.volume}, {"m", tc.mute}} {
			_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(op.key)})
			if cmd == nil {
				t.Fatalf("key %s in panel %s: missing action", op.key, tc.key)
			}
			if msg := cmd().(actionMsg); msg.err != nil {
				t.Fatal(msg.err)
			}
			a := backend.actions[len(backend.actions)-1]
			if a.Operation != op.operation || a.Target != tc.id {
				t.Errorf("key %s in panel %s: %+v", op.key, tc.key, a)
			}
		}
	}
	m := sampleModel()
	m.backend = backend
	m = key(m, "5")
	if m.focus != configuration || m.volumeAction("+5%") != nil || m.muteAction() != nil || m.defaultAction() != nil {
		t.Fatal("configuration has unexpected volume, mute, or default control")
	}
}

func TestActionsUseBackend(t *testing.T) {
	backend := &stubBackend{}
	m := sampleModel()
	m.backend = backend
	m.focus = outputs
	m.cursor[outputs] = 1
	for _, tc := range []struct {
		key  string
		want audio.Action
	}{
		{"m", audio.Action{Operation: audio.ToggleOutputMute, Target: "headphones"}},
		{"d", audio.Action{Operation: audio.SetDefaultOutput, Target: "headphones"}},
		{"+", audio.Action{Operation: audio.SetOutputVolume, Target: "headphones", Value: "+5%"}},
	} {
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)})
		m = updated.(model)
		if cmd == nil {
			t.Fatalf("%s: expected command", tc.key)
		}
		msg := cmd().(actionMsg)
		if msg.err != nil || backend.actions[len(backend.actions)-1] != tc.want {
			t.Errorf("%s: action=%+v, err=%v", tc.key, backend.actions, msg.err)
		}
	}
	m.focus = playback
	m.openMenu()
	if len(m.menu) != 2 || m.menu[0].action.Operation != audio.MovePlayback {
		t.Fatalf("playback menu: %+v", m.menu)
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if len(m.menu) != 0 || cmd == nil {
		t.Fatal("menu selection did not dispatch")
	}
	_ = cmd().(actionMsg)
	if got := backend.actions[len(backend.actions)-1]; got != (audio.Action{Operation: audio.MovePlayback, Target: "4", Value: "speakers"}) {
		t.Errorf("move action = %+v", got)
	}
}
