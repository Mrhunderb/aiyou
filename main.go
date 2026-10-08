package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type sink struct {
	Index       uint32            `json:"index"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Properties  map[string]string `json:"properties"`
	DisplayName string            `json:"-"`
	Mute        bool              `json:"mute"`
	Volume      map[string]struct {
		ValuePercent string `json:"value_percent"`
	} `json:"volume"`
}
type stream struct {
	Index  uint32 `json:"index"`
	Sink   uint32 `json:"sink"`
	Mute   bool   `json:"mute"`
	Volume map[string]struct {
		ValuePercent string `json:"value_percent"`
	} `json:"volume"`
	Properties map[string]string `json:"properties"`
}
type snapshot struct {
	sinks       []sink
	streams     []stream
	defaultSink string
}
type refreshMsg struct {
	data snapshot
	err  error
}
type actionMsg struct {
	text string
	err  error
}
type model struct {
	data                     snapshot
	focus                    int // 0 = outputs, 1 = application streams
	sinkCursor, streamCursor int
	width, height            int
	status                   string
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

func sinkDisplayName(s sink, lookup func(string) string) string {
	if usableName(s.Description) {
		return s.Description
	}
	for _, key := range []string{"device.alias", "device.description", "node.nick"} {
		if name := s.Properties[key]; usableName(name) {
			return name
		}
	}
	if s.Properties["device.bus"] == "bluetooth" {
		address := s.Properties["api.bluez5.address"]
		if name := lookup(address); usableName(name) {
			return name
		}
		if bluetoothAddress.MatchString(address) {
			return "Bluetooth " + address
		}
	}
	return s.Name
}

func loadSnapshot() (snapshot, error) {
	var s snapshot
	b, err := pactl("-f", "json", "list", "sinks")
	if err != nil {
		return s, err
	}
	if err = json.Unmarshal(b, &s.sinks); err != nil {
		return s, fmt.Errorf("decode sinks: %w", err)
	}
	for i := range s.sinks {
		s.sinks[i].DisplayName = sinkDisplayName(s.sinks[i], bluetoothName)
	}
	b, err = pactl("-f", "json", "list", "sink-inputs")
	if err != nil {
		return s, err
	}
	if err = json.Unmarshal(b, &s.streams); err != nil {
		return s, fmt.Errorf("decode streams: %w", err)
	}
	b, err = pactl("get-default-sink")
	if err != nil {
		return s, err
	}
	s.defaultSink = strings.TrimSpace(string(b))
	return s, nil
}
func refresh() tea.Cmd { return func() tea.Msg { s, err := loadSnapshot(); return refreshMsg{s, err} } }
func runAction(args ...string) tea.Cmd {
	return func() tea.Msg { _, err := pactl(args...); return actionMsg{strings.Join(args, " "), err} }
}
func tick() tea.Cmd           { return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return refreshMsg{} }) }
func (m model) Init() tea.Cmd { return tea.Batch(refresh(), tick()) }
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case refreshMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
		} else if msg.data.sinks != nil {
			m.data = msg.data
			if m.sinkCursor >= len(m.data.sinks) {
				m.sinkCursor = max(0, len(m.data.sinks)-1)
			}
			if m.streamCursor >= len(m.data.streams) {
				m.streamCursor = max(0, len(m.data.streams)-1)
			}
		}
		return m, tick()
	case actionMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.status = "Ran: " + msg.text
			return m, refresh()
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.focus = 1 - m.focus
		case "up", "k":
			if m.focus == 0 {
				m.sinkCursor = max(0, m.sinkCursor-1)
			} else {
				m.streamCursor = max(0, m.streamCursor-1)
			}
		case "down", "j":
			if m.focus == 0 {
				m.sinkCursor = min(len(m.data.sinks)-1, m.sinkCursor+1)
			} else {
				m.streamCursor = min(len(m.data.streams)-1, m.streamCursor+1)
			}
		case "d", "enter":
			if m.focus == 0 && len(m.data.sinks) > 0 {
				return m, runAction("set-default-sink", m.data.sinks[m.sinkCursor].Name)
			}
			if m.focus == 1 && len(m.data.streams) > 0 && len(m.data.sinks) > 0 {
				return m, runAction("move-sink-input", strconv.Itoa(int(m.data.streams[m.streamCursor].Index)), m.data.sinks[m.sinkCursor].Name)
			}
		case "+", "=":
			if m.focus == 0 && len(m.data.sinks) > 0 {
				return m, runAction("set-sink-volume", m.data.sinks[m.sinkCursor].Name, "+5%")
			}
		case "-":
			if m.focus == 0 && len(m.data.sinks) > 0 {
				return m, runAction("set-sink-volume", m.data.sinks[m.sinkCursor].Name, "-5%")
			}
		case "m":
			if m.focus == 0 && len(m.data.sinks) > 0 {
				return m, runAction("set-sink-mute", m.data.sinks[m.sinkCursor].Name, "toggle")
			}
			if m.focus == 1 && len(m.data.streams) > 0 {
				return m, runAction("set-sink-input-mute", strconv.Itoa(int(m.data.streams[m.streamCursor].Index)), "toggle")
			}
		case "r":
			return m, refresh()
		}
	}
	return m, nil
}
func percent(v map[string]struct {
	ValuePercent string `json:"value_percent"`
}) string {
	for _, ch := range v {
		return ch.ValuePercent
	}
	return "?"
}
func (m model) View() string {
	var b strings.Builder
	b.WriteString("aiyou — PulseAudio / PipeWire\n\n")
	b.WriteString("OUTPUTS (↑↓ select, d/enter default, +/- volume, m mute)\n")
	for i, s := range m.data.sinks {
		mark := " "
		if i == m.sinkCursor && m.focus == 0 {
			mark = ">"
		}
		def := " "
		if s.Name == m.data.defaultSink {
			def = "*"
		}
		fmt.Fprintf(&b, "%s%s %-42s %4s  %s\n", mark, def, truncate(s.DisplayName, 42), percent(s.Volume), mute(s.Mute))
	}
	b.WriteString("\nAPPLICATION STREAMS (tab focus, d/enter moves to selected output, m mute)\n")
	if len(m.data.streams) == 0 {
		b.WriteString("  (no active streams)\n")
	}
	for i, s := range m.data.streams {
		mark := " "
		if i == m.streamCursor && m.focus == 1 {
			mark = ">"
		}
		name := s.Properties["application.name"]
		if name == "" {
			name = s.Properties["media.name"]
		}
		if name == "" {
			name = fmt.Sprintf("Stream %d", s.Index)
		}
		fmt.Fprintf(&b, "%s #%-5d %-32s %4s  %s\n", mark, s.Index, truncate(name, 32), percent(s.Volume), mute(s.Mute))
	}
	fmt.Fprintf(&b, "\nSelected output: %s\n", selectedSink(m.data, m.sinkCursor))
	if m.status != "" {
		fmt.Fprintf(&b, "%s\n", m.status)
	}
	b.WriteString("\nq quit • tab switch section • r refresh")
	return b.String()
}
func selectedSink(s snapshot, i int) string {
	if i >= 0 && i < len(s.sinks) {
		return s.sinks[i].DisplayName
	}
	return "(none)"
}
func mute(v bool) string {
	if v {
		return "MUTED"
	}
	return "      "
}
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n-1]) + "…"
	}
	return s
}
func main() {
	if _, err := exec.LookPath("pactl"); err != nil {
		fmt.Fprintln(os.Stderr, "aiyou requires pactl (PulseAudio or pipewire-pulse)")
		os.Exit(1)
	}
	p := tea.NewProgram(model{}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
