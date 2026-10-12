package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/Mrhunderb/ayou/internal/audio"
	"github.com/Mrhunderb/ayou/internal/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestConfigChangesControlsAndRendering(t *testing.T) {
	cfg := config.Default()
	cfg.RefreshInterval = config.Duration(250 * time.Millisecond)
	cfg.VolumeStep = 12
	cfg.Theme.Active = "#112233"
	cfg.Theme.SelectionBackground = "#112233"
	cfg.Layout.VolumeBarWidth = 8
	cfg.Layout.CompactBarWidth = 3
	cfg.Layout.CompactBarThreshold = 42
	cfg.Layout.DashboardMinWidth = 120
	cfg.Layout.DashboardMinHeight = 28
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	backend := &stubBackend{}
	m := New(backend, cfg).(model)
	if m.config.RefreshInterval.Duration() != 250*time.Millisecond || string(m.theme.active) != "#112233" {
		t.Fatalf("config not injected: %+v", m.config)
	}
	m.data = sampleModel().data
	m.width, m.height = 100, 30
	if strings.Contains(ansi.Strip(m.View()), "3 PLAYBACK") {
		t.Fatal("custom dashboard minimum width was ignored")
	}
	m.width = 120
	if !strings.Contains(ansi.Strip(m.View()), "3 PLAYBACK") {
		t.Fatal("dashboard not enabled at configured width")
	}
	m.width = 100
	m.help = true
	if !strings.Contains(ansi.Strip(m.View()), "±12%") {
		t.Fatal("help still displays fixed volume step")
	}
	m.help = false
	m.focus = outputs
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})
	if cmd == nil {
		t.Fatal("missing volume action")
	}
	if msg := cmd().(actionMsg); msg.err != nil {
		t.Fatal(msg.err)
	}
	if got := backend.actions[0]; got != (audio.Action{Operation: audio.SetOutputVolume, Target: "speakers", Value: "+12%"}) {
		t.Fatalf("volume step not applied: %+v", got)
	}
	if n := countBar(m.row(outputs, 0, 50)); n != 8 {
		t.Fatalf("normal bar width = %d", n)
	}
	if n := countBar(m.row(outputs, 0, 36)); n != 3 {
		t.Fatalf("compact bar width = %d", n)
	}

	// Bar widths are capped at runtime when a panel is narrower than configured.
	cfg.Layout.VolumeBarWidth, cfg.Layout.CompactBarWidth = 30, 30
	cfg.Layout.CompactBarThreshold = 20
	m = New(backend, cfg).(model)
	m.data = sampleModel().data
	if got := ansi.StringWidth(m.row(outputs, 0, 18)); got > 18 {
		t.Fatalf("bar overflows narrow row: %d", got)
	}
}
func countBar(row string) int {
	s := ansi.Strip(row)
	return strings.Count(s, "━") + strings.Count(s, "─")
}
