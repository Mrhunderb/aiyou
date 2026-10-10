package ui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Mrhunderb/aiyou/internal/audio"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func sampleModel() model {
	v := map[string]audio.ChannelVolume{"front-left": {ValuePercent: "35%", Value: 22934}}
	return model{
		width: 100, height: 30,
		data: audio.Snapshot{
			Sinks:         []audio.Sink{{Index: 1, Name: "speakers", DisplayName: "Speakers", Volume: v}, {Index: 2, Name: "headphones", DisplayName: "耳机 Polaris", Volume: v}},
			Sources:       []audio.Source{{Index: 3, Name: "microphone", DisplayName: "Microphone", Volume: v}},
			SinkInputs:    []audio.SinkInput{{Index: 4, Sink: 1, DisplayName: "Music Player", Volume: v}},
			SourceOutputs: []audio.SourceOutput{{Index: 5, Source: 3, DisplayName: "Recorder", Volume: v}},
			Cards:         []audio.Card{{Name: "card", DisplayName: "Internal Card", ActiveProfile: "HiFi", Profiles: map[string]audio.CardProfile{"off": {Description: "Off", Available: true}}}},
		},
	}
}
func key(m model, name string) model {
	var msg tea.KeyMsg
	switch name {
	case "tab":
		msg = tea.KeyMsg{Type: tea.KeyTab}
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)}
	}
	updated, _ := m.Update(msg)
	return updated.(model)
}
func TestDefaultDeviceMarker(t *testing.T) {
	m := sampleModel()
	m.data.DefaultSink = "headphones"
	m.data.DefaultSource = "microphone"
	m.data.Sinks[1].DisplayName = strings.Repeat("Very Long Headphones ", 5)
	for _, tc := range []struct{ panel, defaultRow int }{{outputs, 1}, {inputs, 0}} {
		m.focus = tc.panel
		m.cursor[tc.panel] = tc.defaultRow
		panel := ansi.Strip(m.panel(tc.panel, 32, 8))
		if !strings.Contains(panel, "› ★ ") {
			t.Errorf("panel %d: selected default marker missing after truncation: %q", tc.panel, panel)
		}
		m.focus = playback // Render the same row without selection.
		panel = ansi.Strip(m.panel(tc.panel, 32, 8))
		if !strings.Contains(panel, "★ ") || strings.Count(panel, "★") != 1 {
			t.Errorf("panel %d: expected exactly one unselected default marker: %q", tc.panel, panel)
		}
	}
	m.data.DefaultSink = "speakers"
	m.focus = outputs
	m.cursor[outputs] = 0
	if !strings.Contains(ansi.Strip(m.panel(outputs, 32, 8)), "› ★ Speakers") {
		t.Fatal("default marker did not follow the changed default device")
	}
	m.data.DefaultSink = ""
	if strings.Contains(ansi.Strip(m.panel(outputs, 32, 8)), "★") {
		t.Fatal("missing default must not mark any device")
	}
}

func TestNavigationAndMenu(t *testing.T) {
	m := sampleModel()
	m = key(m, "1")
	if m.focus != outputs {
		t.Fatalf("focus = %d", m.focus)
	}
	m = key(m, "down")
	if m.selectedID(outputs) != "headphones" {
		t.Fatalf("selection = %q", m.selectedID(outputs))
	}
	m = key(m, "enter")
	if len(m.menu) != 0 || m.status == "" {
		t.Fatalf("expected missing-port status")
	}
	m = key(m, "5")
	m = key(m, "enter")
	if len(m.menu) != 1 || m.menu[0].action.Operation != audio.SetCardProfile {
		t.Fatalf("profile menu: %+v", m.menu)
	}
	m = key(m, "esc")
	if len(m.menu) != 0 {
		t.Fatal("menu not dismissed")
	}
}
func TestDirectionalWidgetFocus(t *testing.T) {
	// Each entry is the expected target for up, down, left, right.
	want := [panelCount][4]int{
		outputs:       {outputs, inputs, outputs, playback},
		inputs:        {outputs, inputs, inputs, configuration},
		playback:      {playback, recording, outputs, playback},
		recording:     {playback, configuration, inputs, recording},
		configuration: {recording, configuration, inputs, configuration},
	}
	keys := [][4]tea.KeyMsg{
		{{Type: tea.KeyCtrlUp}, {Type: tea.KeyCtrlDown}, {Type: tea.KeyCtrlLeft}, {Type: tea.KeyCtrlRight}},
		{{Type: tea.KeyShiftUp}, {Type: tea.KeyShiftDown}, {Type: tea.KeyShiftLeft}, {Type: tea.KeyShiftRight}},
		{{Type: tea.KeyRunes, Runes: []rune("K")}, {Type: tea.KeyRunes, Runes: []rune("J")}, {Type: tea.KeyRunes, Runes: []rune("H")}, {Type: tea.KeyRunes, Runes: []rune("L")}},
		{{Type: tea.KeyRunes, Runes: []rune("W")}, {Type: tea.KeyRunes, Runes: []rune("S")}, {Type: tea.KeyRunes, Runes: []rune("A")}, {Type: tea.KeyRunes, Runes: []rune("D")}},
	}
	for from := 0; from < panelCount; from++ {
		for _, family := range keys {
			for direction, msg := range family {
				m := sampleModel()
				m.focus = from
				updated, _ := m.Update(msg)
				if got := updated.(model).focus; got != want[from][direction] {
					t.Errorf("from %d, %s: focus = %d, want %d", from, msg.String(), got, want[from][direction])
				}
			}
		}
	}
	m := sampleModel()
	m.focus = outputs
	m = key(m, "down")
	if m.focus != outputs || m.cursor[outputs] != 1 {
		t.Fatalf("ordinary down should only select rows: focus=%d cursor=%d", m.focus, m.cursor[outputs])
	}
}

func TestExpandedWidgetLocksFocus(t *testing.T) {
	m := sampleModel()
	m = key(m, "e")
	if !m.expanded {
		t.Fatal("widget did not expand")
	}
	for _, msg := range []tea.KeyMsg{
		{Type: tea.KeyTab}, {Type: tea.KeyShiftTab},
		{Type: tea.KeyRunes, Runes: []rune("2")}, {Type: tea.KeyRunes, Runes: []rune("3")},
		{Type: tea.KeyRunes, Runes: []rune("4")}, {Type: tea.KeyRunes, Runes: []rune("5")},
		{Type: tea.KeyCtrlDown}, {Type: tea.KeyShiftRight},
		{Type: tea.KeyRunes, Runes: []rune("J")}, {Type: tea.KeyRunes, Runes: []rune("D")},
	} {
		updated, _ := m.Update(msg)
		m = updated.(model)
		if m.focus != outputs || !m.expanded {
			t.Fatalf("%s switched expanded widget: focus=%d", msg.String(), m.focus)
		}
	}
	m = key(m, "down")
	if m.cursor[outputs] != 1 {
		t.Fatal("row navigation was disabled in expanded mode")
	}
	m = key(m, "e")
	m = key(m, "tab")
	if m.expanded || m.focus != inputs {
		t.Fatalf("focus did not unlock after e: %+v", m)
	}
	m = key(m, "e")
	m = key(m, "esc")
	m = key(m, "tab")
	if m.expanded || m.focus != playback {
		t.Fatalf("focus did not unlock after Esc: %+v", m)
	}
}

func TestFivePanelDashboard(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {80, 24}} {
		m := sampleModel()
		m.width, m.height = size[0], size[1]
		m.focus = configuration // Configuration should stay in the dashboard.
		lines := strings.Split(ansi.Strip(m.View()), "\n")
		positions := map[string]struct{ line, col int }{}
		for row, line := range lines {
			for _, title := range []string{"1 OUTPUT DEVICES", "2 INPUT DEVICES", "3 PLAYBACK", "4 RECORDING", "5 CONFIGURATION"} {
				if col := strings.Index(line, title); col >= 0 {
					positions[title] = struct{ line, col int }{row, col}
				}
			}
		}
		if len(positions) != 5 {
			t.Fatalf("%dx%d: expected five visible widgets, got %v", m.width, m.height, positions)
		}
		o, i := positions["1 OUTPUT DEVICES"], positions["2 INPUT DEVICES"]
		p, r, c := positions["3 PLAYBACK"], positions["4 RECORDING"], positions["5 CONFIGURATION"]
		if !(o.col == i.col && p.col == r.col && r.col == c.col && o.col < p.col && o.line < i.line && p.line < r.line && r.line < c.line) {
			t.Errorf("%dx%d: wrong grid positions: %v", m.width, m.height, positions)
		}
	}
}

func TestRenderBounds(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {80, 24}, {60, 18}, {20, 9}} {
		m := sampleModel()
		m.width, m.height = size[0], size[1]
		for _, mode := range []string{"dashboard", "expanded", "help", "menu"} {
			m.expanded, m.help, m.menu = false, false, nil
			switch mode {
			case "expanded":
				m.expanded = true
			case "help":
				m.help = true
			case "menu":
				m.menuTitle = "PROFILES"
				for i := 0; i < 30; i++ {
					m.menu = append(m.menu, menuOption{label: "Profile"})
				}
				m.menuCursor = 29
			}
			view := m.View()
			lines := strings.Split(view, "\n")
			if len(lines) > m.height {
				t.Errorf("%dx%d %s: view has %d lines", m.width, m.height, mode, len(lines))
			}
			for i, line := range lines {
				if lipgloss.Width(line) > m.width {
					t.Errorf("%dx%d %s: line %d width %d", m.width, m.height, mode, i, lipgloss.Width(line))
				}
			}
		}
	}
}
func TestTruncateCells(t *testing.T) {
	if lipgloss.Width(truncate("中文耳机 Polaris", 8)) > 8 {
		t.Fatal("unicode truncate exceeds width")
	}
	got := truncate(strings.Repeat("中文", 25), 42)
	if !utf8.ValidString(got) || !strings.HasSuffix(got, "…") {
		t.Fatalf("invalid truncated display: %q", got)
	}
}
