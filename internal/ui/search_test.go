package ui

import (
	"strings"
	"testing"

	"github.com/Mrhunderb/aiyou/internal/audio"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func searchModel() model {
	m := sampleModel()
	m.data.Sinks = []audio.Sink{
		{Name: "speakers", DisplayName: "Speakers"},
		{Name: "headphones", DisplayName: "Headphones"},
		{Name: "display", DisplayName: "Display"},
		{Name: "headset", DisplayName: "HEADSET"},
	}
	return m
}
func TestSearchFiltersRowsAndKeepsOriginalTargets(t *testing.T) {
	backend := &stubBackend{}
	m := searchModel()
	m.backend = backend
	m = key(m, "/")
	m = key(m, "HeAd")
	if !m.searching || m.selectedID(outputs) != "headphones" {
		t.Fatalf("typing should select first matching row: cursor=%d id=%q", m.cursor[outputs], m.selectedID(outputs))
	}
	panel := ansi.Strip(m.panel(outputs, 50, 15))
	for _, want := range []string{"Headphones", "HEADSET"} {
		if !strings.Contains(panel, want) {
			t.Errorf("missing matching row %q in %q", want, panel)
		}
	}
	for _, hidden := range []string{"Speakers", "Display"} {
		if strings.Contains(panel, hidden) {
			t.Errorf("non-matching row %q is still shown", hidden)
		}
	}
	if other := ansi.Strip(m.panel(inputs, 50, 15)); !strings.Contains(other, "Microphone") {
		t.Fatal("search filtered an unfocused widget")
	}
	m = key(m, "down") // Arrow keys navigate filtered rows even while typing.
	if m.selectedID(outputs) != "headset" {
		t.Fatal("down while typing did not select next match")
	}
	updatedWhileTyping, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updatedWhileTyping.(model)
	if m.selectedID(outputs) != "headphones" {
		t.Fatal("up while typing did not select previous match")
	}
	m = key(m, "enter") // Finish editing; keep filtered rows.
	m = key(m, "down")
	if m.selectedID(outputs) != "headset" {
		t.Fatalf("down selected %q", m.selectedID(outputs))
	}
	m = key(m, "g")
	if m.selectedID(outputs) != "headphones" {
		t.Fatal("g did not select first match")
	}
	m = key(m, "G")
	if m.selectedID(outputs) != "headset" {
		t.Fatal("G did not select last match")
	}
	m = key(m, "n")
	if m.selectedID(outputs) != "headphones" {
		t.Fatal("n did not cycle matches")
	}
	m = key(m, "N")
	if m.selectedID(outputs) != "headset" {
		t.Fatal("N did not cycle matches")
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = updated.(model)
	if cmd == nil {
		t.Fatal("no action for selected filtered row")
	}
	if msg := cmd().(actionMsg); msg.err != nil {
		t.Fatal(msg.err)
	}
	if got := backend.actions[len(backend.actions)-1]; got.Target != "headset" || got.Operation != audio.SetDefaultOutput {
		t.Fatalf("action targeted wrong snapshot row: %+v", got)
	}
	m = key(m, "esc") // Clear filter and show all rows again.
	if m.search != "" || len(m.visibleIndices(outputs)) != 4 {
		t.Fatal("Esc did not clear filter")
	}
}
func TestSearchNoMatchesDisablesActions(t *testing.T) {
	backend := &stubBackend{}
	m := searchModel()
	m.backend = backend
	m = key(m, "/")
	m = key(m, "notfound")
	if m.selectedID(outputs) != "" {
		t.Fatal("hidden row remains selected")
	}
	if !strings.Contains(ansi.Strip(m.panel(outputs, 50, 15)), "(no matches)") {
		t.Fatal("missing empty search state")
	}
	m = key(m, "enter")
	for _, name := range []string{"m", "d", "+", "enter"} {
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)})
		m = updated.(model)
		if cmd != nil || len(m.menu) != 0 {
			t.Fatalf("%s triggered hidden item action", name)
		}
	}
	if len(backend.actions) != 0 {
		t.Fatal("backend was called for hidden item")
	}
	m = key(m, "/")
	m = key(m, "headx")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(model)
	if m.search != "head" || m.selectedID(outputs) != "headphones" {
		t.Fatalf("backspace failed to restore matches: search=%q, id=%q", m.search, m.selectedID(outputs))
	}
	m = key(m, "esc") // Stop editing without clearing matches.
	if m.search != "head" || m.searching {
		t.Fatal("Esc while editing should leave filter in place")
	}
	m = key(m, "tab") // Switching widgets clears the focused filter.
	if m.search != "" || m.focus != inputs {
		t.Fatal("focus change failed to clear filter")
	}
}
func TestFilteredRefreshMaintainsSelection(t *testing.T) {
	backend := &stubBackend{snapshot: searchModel().data}
	m := New(backend).(model)
	updated, _ := m.Update(m.refresh()())
	m = updated.(model)
	m.search = "head"
	m.normalizeSelection()
	m.move(1) // HEADSET
	backend.snapshot.Sinks = []audio.Sink{
		{Name: "headset", DisplayName: "HEADSET"},
		{Name: "speakers", DisplayName: "Speakers"},
		{Name: "headphones", DisplayName: "Headphones"},
	}
	updated, _ = m.Update(m.refresh()())
	m = updated.(model)
	if m.selectedID(outputs) != "headset" || m.cursor[outputs] != 0 {
		t.Fatalf("selection lost on reorder: %q", m.selectedID(outputs))
	}
	backend.snapshot.Sinks = []audio.Sink{{Name: "speakers", DisplayName: "Speakers"}, {Name: "headphones", DisplayName: "Headphones"}}
	updated, _ = m.Update(m.refresh()())
	m = updated.(model)
	if m.selectedID(outputs) != "headphones" {
		t.Fatalf("should fall back to first match: %q", m.selectedID(outputs))
	}
	backend.snapshot.Sinks = []audio.Sink{{Name: "speakers", DisplayName: "Speakers"}}
	updated, _ = m.Update(m.refresh()())
	m = updated.(model)
	if m.selectedID(outputs) != "" {
		t.Fatal("no-match refresh retained hidden selection")
	}
	backend.snapshot.Sinks = []audio.Sink{{Name: "headphones", DisplayName: "Headphones"}}
	updated, _ = m.Update(m.refresh()())
	m = updated.(model)
	if m.selectedID(outputs) != "headphones" {
		t.Fatal("new match not selected after refresh")
	}
}
