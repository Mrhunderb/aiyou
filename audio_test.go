package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A fake pactl binary exercises JSON parsing and action command construction
// without changing the real audio server.
func fakePactl(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	script := `#!/bin/sh
printf '%s\n' "$*" >> "` + log + `"
case "$*" in
  '-f json list sinks') printf '%s\n' '[{"index":1,"name":"speaker","description":"Speaker","volume":{"mono":{"value_percent":"35%"}},"ports":[{"name":"analog","description":"Analog"}]}]' ;;
  '-f json list sources') printf '%s\n' '[{"index":3,"name":"speaker.monitor","description":"Monitor","properties":{"device.class":"monitor"}},{"index":5,"name":"mic","description":"Mic"}]' ;;
  '-f json list sink-inputs') printf '%s\n' '[{"index":4,"sink":1,"properties":{"application.name":"Music"}}]' ;;
  '-f json list source-outputs') printf '%s\n' '[{"index":6,"source":5,"properties":{"application.name":"Recorder"}}]' ;;
  '-f json list cards') printf '%s\n' '[{"index":7,"name":"card","properties":{"device.description":"Sound Card"},"profiles":{"off":{"description":"Off","available":true}},"active_profile":"off"}]' ;;
  'get-default-sink') echo speaker ;;
  'get-default-source') echo mic ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "pactl"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}
func TestLoadSnapshotAndCommands(t *testing.T) {
	log := fakePactl(t)
	s, err := loadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Sinks) != 1 || len(s.Sources) != 2 || !s.Sources[0].IsMonitor ||
		len(s.SinkInputs) != 1 || s.SinkInputs[0].DisplayName != "Music" ||
		len(s.SourceOutputs) != 1 || len(s.Cards) != 1 || s.DefaultSink != "speaker" || s.DefaultSource != "mic" {
		t.Fatalf("unexpected snapshot: %+v", s)
	}
	m := model{data: s, focus: playback}
	if msg := m.volumeAction("+5%")().(actionMsg); msg.err != nil {
		t.Fatal(msg.err)
	}
	m.focus = recording
	if msg := m.muteAction()().(actionMsg); msg.err != nil {
		t.Fatal(msg.err)
	}
	m.focus = inputs
	m.cursor[inputs] = 1
	if msg := m.defaultAction()().(actionMsg); msg.err != nil {
		t.Fatal(msg.err)
	}
	m.focus = configuration
	m.openMenu()
	if len(m.menu) != 1 || strings.Join(m.menu[0].args, " ") != "set-card-profile card off" {
		t.Fatalf("profile menu: %+v", m.menu)
	}
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []string{"set-sink-input-volume 4 +5%", "set-source-output-mute 6 toggle", "set-default-source mic"} {
		if !strings.Contains(string(b), cmd) {
			t.Errorf("missing %q in %q", cmd, b)
		}
	}
}
