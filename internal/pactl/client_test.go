package pactl

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Mrhunderb/aiyou/internal/audio"
)

func TestDeviceDisplayName(t *testing.T) {
	props := map[string]string{"device.bus": "bluetooth", "device.alias": "(null)", "api.bluez5.address": "41:42:D1:97:77:91"}
	lookup := func(addr string) string {
		if addr != "41:42:D1:97:77:91" {
			t.Fatalf("unexpected address %q", addr)
		}
		return "觅声-Polaris"
	}
	if got := resolveDeviceDisplayName("(null)", props, "bluez", lookup); got != "觅声-Polaris" {
		t.Fatalf("bluetooth alias = %q", got)
	}
	if got := resolveDeviceDisplayName("Speakers", props, "bluez", nil); got != "Speakers" {
		t.Fatalf("description = %q", got)
	}
	if got := resolveDeviceDisplayName("(null)", props, "bluez", func(string) string { return "" }); got != "Bluetooth 41:42:D1:97:77:91" {
		t.Fatalf("fallback = %q", got)
	}
	if got := resolveDeviceDisplayName("(null)", nil, "bluez", nil); got != "bluez" {
		t.Fatalf("name = %q", got)
	}
}
func TestParseBluetoothName(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{"\tName: 旧名\n\tAlias: 觅声-Polaris\n", "觅声-Polaris"},
		{"\tName: 中文耳机\n", "中文耳机"},
		{"\tAlias: (null)\n", ""},
	} {
		if got := parseBluetoothName(tc.raw); got != tc.want {
			t.Errorf("parse %q = %q", tc.raw, got)
		}
	}
}
func TestSnapshot(t *testing.T) {
	calls := map[string]string{
		"-f json list sinks":          `[{"index":1,"name":"speaker","description":"Speaker","volume":{"mono":{"value_percent":"35%"}}}]`,
		"-f json list sources":        `[{"index":3,"name":"speaker.monitor","description":"Monitor","properties":{"device.class":"monitor"}},{"index":5,"name":"mic","description":"Mic"}]`,
		"-f json list sink-inputs":    `[{"index":4,"sink":1,"properties":{"application.name":"Music"}}]`,
		"-f json list source-outputs": `[{"index":6,"source":5,"properties":{"application.name":"Recorder"}}]`,
		"-f json list cards":          `[{"index":7,"name":"card","properties":{"device.description":"Sound Card"},"profiles":{"off":{"description":"Off","available":true}},"active_profile":"off"}]`,
		"get-default-sink":            "speaker\n",
		"get-default-source":          "mic\n",
	}
	c := &Client{run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "pactl" {
			t.Fatalf("command = %q", name)
		}
		key := strings.Join(args, " ")
		if data, ok := calls[key]; ok {
			return []byte(data), nil
		}
		return nil, fmt.Errorf("unexpected call %q", key)
	}}
	s, err := c.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Sinks) != 1 || len(s.Sources) != 2 || !s.Sources[0].IsMonitor ||
		len(s.SinkInputs) != 1 || s.SinkInputs[0].DisplayName != "Music" ||
		len(s.SourceOutputs) != 1 || len(s.Cards) != 1 || s.DefaultSink != "speaker" || s.DefaultSource != "mic" {
		t.Fatalf("unexpected snapshot: %+v", s)
	}
	c.run = func(_ context.Context, _ string, args ...string) ([]byte, error) { return []byte("not json"), nil }
	if _, err := c.Snapshot(); err == nil || !strings.Contains(err.Error(), "decode sinks") {
		t.Fatalf("expected decode error, got %v", err)
	}
}
func TestArguments(t *testing.T) {
	tests := []struct {
		action audio.Action
		want   []string
	}{
		{audio.Action{Operation: audio.SetPlaybackVolume, Target: "4", Value: "+5%"}, []string{"set-sink-input-volume", "4", "+5%"}},
		{audio.Action{Operation: audio.ToggleRecordingMute, Target: "6"}, []string{"set-source-output-mute", "6", "toggle"}},
		{audio.Action{Operation: audio.SetDefaultInput, Target: "mic"}, []string{"set-default-source", "mic"}},
		{audio.Action{Operation: audio.SetCardProfile, Target: "card", Value: "HiFi (Speaker)"}, []string{"set-card-profile", "card", "HiFi (Speaker)"}},
	}
	for _, tc := range tests {
		got, err := Arguments(tc.action)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %v, %v; want %v", tc.action, got, err, tc.want)
		}
	}
	// Cover every operation so adding a control cannot silently use the wrong pactl verb.
	for op, verb := range map[audio.Operation]string{
		audio.SetPlaybackVolume: "set-sink-input-volume", audio.SetRecordingVolume: "set-source-output-volume",
		audio.SetOutputVolume: "set-sink-volume", audio.SetInputVolume: "set-source-volume",
		audio.TogglePlaybackMute: "set-sink-input-mute", audio.ToggleRecordingMute: "set-source-output-mute",
		audio.ToggleOutputMute: "set-sink-mute", audio.ToggleInputMute: "set-source-mute",
		audio.SetDefaultOutput: "set-default-sink", audio.SetDefaultInput: "set-default-source",
		audio.MovePlayback: "move-sink-input", audio.MoveRecording: "move-source-output",
		audio.SetOutputPort: "set-sink-port", audio.SetInputPort: "set-source-port",
		audio.SetCardProfile: "set-card-profile",
	} {
		args, err := Arguments(audio.Action{Operation: op, Target: "device", Value: "choice"})
		if err != nil || args[0] != verb {
			t.Errorf("%s: args=%v err=%v, want %s", op, args, err, verb)
		}
	}
	for _, action := range []audio.Action{{}, {Operation: audio.SetOutputVolume, Target: "sink"}, {Operation: audio.SetDefaultInput}} {
		if _, err := Arguments(action); err == nil {
			t.Errorf("expected validation error for %+v", action)
		}
	}
}
