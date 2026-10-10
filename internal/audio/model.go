// Package audio defines the state and operations of the audio control application.
// It does not depend on the terminal UI or on pactl.
package audio

import (
	"fmt"
	"strconv"
	"strings"
)

type ChannelVolume struct {
	Value        int    `json:"value"`
	ValuePercent string `json:"value_percent"`
	DB           string `json:"db"`
}
type Port struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Type         string `json:"type"`
	Availability string `json:"availability"`
}
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
type SinkInput struct {
	Index       uint32                   `json:"index"`
	Sink        uint32                   `json:"sink"`
	Mute        bool                     `json:"mute"`
	Volume      map[string]ChannelVolume `json:"volume"`
	Properties  map[string]string        `json:"properties"`
	DisplayName string                   `json:"-"`
}
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

// Snapshot is a point-in-time view returned by the backend.
type Snapshot struct {
	Sinks         []Sink
	Sources       []Source
	SinkInputs    []SinkInput
	SourceOutputs []SourceOutput
	Cards         []Card
	DefaultSink   string
	DefaultSource string
}

// Operation names are domain concepts; only the backend knows the CLI syntax.
type Operation string

const (
	SetPlaybackVolume   Operation = "set-playback-volume"
	SetRecordingVolume  Operation = "set-recording-volume"
	SetOutputVolume     Operation = "set-output-volume"
	SetInputVolume      Operation = "set-input-volume"
	TogglePlaybackMute  Operation = "toggle-playback-mute"
	ToggleRecordingMute Operation = "toggle-recording-mute"
	ToggleOutputMute    Operation = "toggle-output-mute"
	ToggleInputMute     Operation = "toggle-input-mute"
	SetDefaultOutput    Operation = "set-default-output"
	SetDefaultInput     Operation = "set-default-input"
	MovePlayback        Operation = "move-playback"
	MoveRecording       Operation = "move-recording"
	SetOutputPort       Operation = "set-output-port"
	SetInputPort        Operation = "set-input-port"
	SetCardProfile      Operation = "set-card-profile"
)

// Action targets a stream/device/card by its current stable name or index.
// Value is used for volume adjustments, destinations, ports and profiles.
type Action struct {
	Operation Operation
	Target    string
	Value     string
}

func (a Action) String() string {
	if a.Value == "" {
		return fmt.Sprintf("%s %s", a.Operation, a.Target)
	}
	return fmt.Sprintf("%s %s %s", a.Operation, a.Target, a.Value)
}

// Backend makes the UI testable without invoking a real audio server.
type Backend interface {
	Snapshot() (Snapshot, error)
	Execute(Action) error
}

// AverageVolume returns the integer mean across parseable channels, its label
// and a representative dB value. Volume controls still operate on all channels.
func AverageVolume(v map[string]ChannelVolume) (int, string, string) {
	if len(v) == 0 {
		return 0, "0%", ""
	}
	sum, count := 0, 0
	percentStr, dbStr := "0%", ""
	for _, cv := range v {
		p := strings.TrimSuffix(strings.TrimSpace(cv.ValuePercent), "%")
		if val, err := strconv.Atoi(p); err == nil {
			sum += val
			count++
		}
		percentStr, dbStr = cv.ValuePercent, cv.DB
	}
	if count > 0 {
		avg := sum / count
		return avg, fmt.Sprintf("%d%%", avg), dbStr
	}
	return 0, percentStr, dbStr
}
