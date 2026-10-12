// Package pactl implements the audio backend using PulseAudio's pactl command.
package pactl

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Mrhunderb/ayou/internal/audio"
)

// Client invokes pactl to load state and execute audio operations.
type Client struct {
	run     func(context.Context, string, ...string) ([]byte, error)
	lookup  func(string) string
	timeout time.Duration
}

func New(pactlTimeout, bluetoothTimeout time.Duration) *Client {
	return &Client{
		run: command, timeout: pactlTimeout,
		lookup: func(address string) string { return bluetoothName(address, bluetoothTimeout) },
	}
}

func command(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return nil, fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), detail)
	}
	return out, nil
}
func (c *Client) call(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	return c.run(ctx, "pactl", args...)
}

var bluetoothAddress = regexp.MustCompile(`(?i)^[0-9a-f]{2}(:[0-9a-f]{2}){5}$`)

func usableName(name string) bool { return name != "" && name != "(null)" }
func parseBluetoothName(output string) string {
	for _, prefix := range []string{"Alias:", "Name:"} {
		for _, line := range strings.Split(output, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, prefix) {
				name := strings.TrimSpace(strings.TrimPrefix(line, prefix))
				if usableName(name) {
					return name
				}
			}
		}
	}
	return ""
}
func bluetoothName(address string, timeout time.Duration) string {
	if !bluetoothAddress.MatchString(address) {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := command(ctx, "bluetoothctl", "info", address)
	if err != nil {
		return ""
	}
	return parseBluetoothName(string(out))
}
func resolveDeviceDisplayName(desc string, props map[string]string, fallbackName string, lookup func(string) string) string {
	if usableName(desc) {
		return desc
	}
	for _, key := range []string{"device.alias", "device.description", "node.nick", "alsa.card_name"} {
		if name := props[key]; usableName(name) {
			return name
		}
	}
	if props["device.bus"] == "bluetooth" {
		address := props["api.bluez5.address"]
		if lookup != nil {
			if name := lookup(address); usableName(name) {
				return name
			}
		}
		if bluetoothAddress.MatchString(address) {
			return "Bluetooth " + address
		}
	}
	return fallbackName
}
func resolveStreamDisplayName(props map[string]string, index uint32) string {
	for _, key := range []string{"application.name", "media.name", "application.process.binary"} {
		if val := props[key]; usableName(val) {
			return val
		}
	}
	return fmt.Sprintf("Stream #%d", index)
}

// Snapshot reads the five pactl lists, then resolves display names and defaults.
func (c *Client) Snapshot() (audio.Snapshot, error) {
	var s audio.Snapshot
	lists := []struct {
		name string
		dest any
	}{
		{"sinks", &s.Sinks}, {"sources", &s.Sources},
		{"sink-inputs", &s.SinkInputs}, {"source-outputs", &s.SourceOutputs},
		{"cards", &s.Cards},
	}
	for _, list := range lists {
		b, err := c.call("-f", "json", "list", list.name)
		if err != nil {
			return s, fmt.Errorf("list %s: %w", list.name, err)
		}
		if err := json.Unmarshal(b, list.dest); err != nil {
			return s, fmt.Errorf("decode %s: %w", list.name, err)
		}
	}
	for i := range s.Sinks {
		d := &s.Sinks[i]
		d.DisplayName = resolveDeviceDisplayName(d.Description, d.Properties, d.Name, c.lookup)
	}
	for i := range s.Sources {
		d := &s.Sources[i]
		d.DisplayName = resolveDeviceDisplayName(d.Description, d.Properties, d.Name, c.lookup)
		d.IsMonitor = strings.HasSuffix(d.Name, ".monitor") || d.Properties["device.class"] == "monitor"
	}
	for i := range s.SinkInputs {
		d := &s.SinkInputs[i]
		d.DisplayName = resolveStreamDisplayName(d.Properties, d.Index)
	}
	for i := range s.SourceOutputs {
		d := &s.SourceOutputs[i]
		d.DisplayName = resolveStreamDisplayName(d.Properties, d.Index)
	}
	for i := range s.Cards {
		d := &s.Cards[i]
		d.DisplayName = resolveDeviceDisplayName(d.Properties["device.description"], d.Properties, d.Name, c.lookup)
	}
	// Servers without get-default-source support still expose other controls.
	if b, err := c.call("get-default-sink"); err == nil {
		s.DefaultSink = strings.TrimSpace(string(b))
	}
	if b, err := c.call("get-default-source"); err == nil {
		s.DefaultSource = strings.TrimSpace(string(b))
	}
	return s, nil
}

// Arguments maps domain actions to pactl without passing user-supplied values
// through a shell. It rejects unknown operations and missing targets/values.
func Arguments(a audio.Action) ([]string, error) {
	if a.Target == "" {
		return nil, fmt.Errorf("%s: missing target", a.Operation)
	}
	var verb string
	switch a.Operation {
	case audio.SetPlaybackVolume:
		verb = "set-sink-input-volume"
	case audio.SetRecordingVolume:
		verb = "set-source-output-volume"
	case audio.SetOutputVolume:
		verb = "set-sink-volume"
	case audio.SetInputVolume:
		verb = "set-source-volume"
	case audio.TogglePlaybackMute:
		verb = "set-sink-input-mute"
	case audio.ToggleRecordingMute:
		verb = "set-source-output-mute"
	case audio.ToggleOutputMute:
		verb = "set-sink-mute"
	case audio.ToggleInputMute:
		verb = "set-source-mute"
	case audio.SetDefaultOutput:
		verb = "set-default-sink"
	case audio.SetDefaultInput:
		verb = "set-default-source"
	case audio.MovePlayback:
		verb = "move-sink-input"
	case audio.MoveRecording:
		verb = "move-source-output"
	case audio.SetOutputPort:
		verb = "set-sink-port"
	case audio.SetInputPort:
		verb = "set-source-port"
	case audio.SetCardProfile:
		verb = "set-card-profile"
	default:
		return nil, fmt.Errorf("unknown audio operation %q", a.Operation)
	}
	args := []string{verb, a.Target}
	switch a.Operation {
	case audio.TogglePlaybackMute, audio.ToggleRecordingMute, audio.ToggleOutputMute, audio.ToggleInputMute:
		args = append(args, "toggle")
	case audio.SetDefaultOutput, audio.SetDefaultInput:
		// No value required.
	default:
		if a.Value == "" {
			return nil, fmt.Errorf("%s: missing value", a.Operation)
		}
		args = append(args, a.Value)
	}
	return args, nil
}
func (c *Client) Execute(a audio.Action) error {
	args, err := Arguments(a)
	if err != nil {
		return err
	}
	_, err = c.call(args...)
	return err
}
