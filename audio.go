package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ChannelVolume represents volume info for a channel.
type ChannelVolume struct {
	Value        int    `json:"value"`
	ValuePercent string `json:"value_percent"`
	DB           string `json:"db"`
}

// Port represents a physical or virtual port on a device.
type Port struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Type         string `json:"type"`
	Availability string `json:"availability"`
}

// Sink represents an output audio device.
type Sink struct {
	Index       uint32                   `json:"index"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Properties  map[string]string        `json:"properties"`
	DisplayName string                   `json:"-"`
	Mute        bool                     `json:"mute"`
	Volume      map[string]ChannelVolume `json:"volume"`
	ActivePort  string                   `json:"active_port"`
	Ports       []Port                   `json:"ports"`
}

// Source represents an input audio device (microphone or monitor).
type Source struct {
	Index       uint32                   `json:"index"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Properties  map[string]string        `json:"properties"`
	DisplayName string                   `json:"-"`
	Mute        bool                     `json:"mute"`
	Volume      map[string]ChannelVolume `json:"volume"`
	ActivePort  string                   `json:"active_port"`
	Ports       []Port                   `json:"ports"`
	IsMonitor   bool                     `json:"-"`
}

// SinkInput represents an active application playback audio stream.
type SinkInput struct {
	Index       uint32                   `json:"index"`
	Sink        uint32                   `json:"sink"`
	Mute        bool                     `json:"mute"`
	Volume      map[string]ChannelVolume `json:"volume"`
	Properties  map[string]string        `json:"properties"`
	DisplayName string                   `json:"-"`
}

// SourceOutput represents an active application recording stream.
type SourceOutput struct {
	Index       uint32                   `json:"index"`
	Source      uint32                   `json:"source"`
	Mute        bool                     `json:"mute"`
	Volume      map[string]ChannelVolume `json:"volume"`
	Properties  map[string]string        `json:"properties"`
	DisplayName string                   `json:"-"`
}

type CardProfile struct {
	Description string `json:"description"`
	Available   bool   `json:"available"`
}

type Card struct {
	Index         uint32                 `json:"index"`
	Name          string                 `json:"name"`
	Properties    map[string]string      `json:"properties"`
	Profiles      map[string]CardProfile `json:"profiles"`
	ActiveProfile string                 `json:"active_profile"`
	DisplayName   string                 `json:"-"`
}

// AudioSnapshot represents a point-in-time state of the audio system.
type AudioSnapshot struct {
	Sinks         []Sink
	Sources       []Source
	SinkInputs    []SinkInput
	SourceOutputs []SourceOutput
	Cards         []Card
	DefaultSink   string
	DefaultSource string
}

func pactl(args ...string) ([]byte, error) {
	cmd := exec.Command("pactl", args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return nil, fmt.Errorf("pactl %s: %s", strings.Join(args, " "), detail)
	}
	return out, nil
}

var bluetoothAddress = regexp.MustCompile(`(?i)^[0-9a-f]{2}(:[0-9a-f]{2}){5}$`)

func usableName(name string) bool {
	return name != "" && name != "(null)"
}

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

func bluetoothName(address string) string {
	if !bluetoothAddress.MatchString(address) {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "bluetoothctl", "info", address).Output()
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

// Retain the original helper used by existing tests and clients.
func sinkDisplayName(s Sink, lookup func(string) string) string {
	return resolveDeviceDisplayName(s.Description, s.Properties, s.Name, lookup)
}

func resolveStreamDisplayName(props map[string]string, index uint32) string {
	for _, key := range []string{"application.name", "media.name", "application.process.binary"} {
		if val := props[key]; usableName(val) {
			return val
		}
	}
	return fmt.Sprintf("Stream #%d", index)
}

func getAvgVolume(v map[string]ChannelVolume) (int, string, string) {
	if len(v) == 0 {
		return 0, "0%", ""
	}
	sum := 0
	count := 0
	percentStr := "0%"
	dbStr := ""
	for _, cv := range v {
		p := strings.TrimSuffix(strings.TrimSpace(cv.ValuePercent), "%")
		if val, err := strconv.Atoi(p); err == nil {
			sum += val
			count++
		}
		percentStr = cv.ValuePercent
		dbStr = cv.DB
	}
	if count > 0 {
		avg := sum / count
		return avg, fmt.Sprintf("%d%%", avg), dbStr
	}
	return 0, percentStr, dbStr
}

func loadSnapshot() (AudioSnapshot, error) {
	var s AudioSnapshot

	// 1. Sinks
	b, err := pactl("-f", "json", "list", "sinks")
	if err != nil {
		return s, fmt.Errorf("list sinks: %w", err)
	}
	if err = json.Unmarshal(b, &s.Sinks); err != nil {
		return s, fmt.Errorf("decode sinks: %w", err)
	}
	for i := range s.Sinks {
		s.Sinks[i].DisplayName = resolveDeviceDisplayName(
			s.Sinks[i].Description,
			s.Sinks[i].Properties,
			s.Sinks[i].Name,
			bluetoothName,
		)
	}

	// 2. Sources
	b, err = pactl("-f", "json", "list", "sources")
	if err != nil {
		return s, fmt.Errorf("list sources: %w", err)
	}
	if err = json.Unmarshal(b, &s.Sources); err != nil {
		return s, fmt.Errorf("decode sources: %w", err)
	}
	for i := range s.Sources {
		s.Sources[i].DisplayName = resolveDeviceDisplayName(
			s.Sources[i].Description,
			s.Sources[i].Properties,
			s.Sources[i].Name,
			bluetoothName,
		)
		isMon := strings.HasSuffix(s.Sources[i].Name, ".monitor") ||
			s.Sources[i].Properties["device.class"] == "monitor"
		s.Sources[i].IsMonitor = isMon
	}

	// 3. Playback streams (sink-inputs)
	b, err = pactl("-f", "json", "list", "sink-inputs")
	if err != nil {
		return s, fmt.Errorf("list sink-inputs: %w", err)
	}
	if err = json.Unmarshal(b, &s.SinkInputs); err != nil {
		return s, fmt.Errorf("decode sink-inputs: %w", err)
	}
	for i := range s.SinkInputs {
		s.SinkInputs[i].DisplayName = resolveStreamDisplayName(s.SinkInputs[i].Properties, s.SinkInputs[i].Index)
	}

	// 4. Recording streams (source-outputs)
	b, err = pactl("-f", "json", "list", "source-outputs")
	if err != nil {
		return s, fmt.Errorf("list source-outputs: %w", err)
	}
	if err = json.Unmarshal(b, &s.SourceOutputs); err != nil {
		return s, fmt.Errorf("decode source-outputs: %w", err)
	}
	for i := range s.SourceOutputs {
		s.SourceOutputs[i].DisplayName = resolveStreamDisplayName(s.SourceOutputs[i].Properties, s.SourceOutputs[i].Index)
	}

	// 5. Cards / profiles
	b, err = pactl("-f", "json", "list", "cards")
	if err != nil {
		return s, fmt.Errorf("list cards: %w", err)
	}
	if err = json.Unmarshal(b, &s.Cards); err != nil {
		return s, fmt.Errorf("decode cards: %w", err)
	}
	for i := range s.Cards {
		s.Cards[i].DisplayName = resolveDeviceDisplayName(s.Cards[i].Properties["device.description"], s.Cards[i].Properties, s.Cards[i].Name, bluetoothName)
	}

	// 6. Default Sink & Source
	if b, err = pactl("get-default-sink"); err == nil {
		s.DefaultSink = strings.TrimSpace(string(b))
	}
	if b, err = pactl("get-default-source"); err == nil {
		s.DefaultSource = strings.TrimSpace(string(b))
	}

	return s, nil
}
