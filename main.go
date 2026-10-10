package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	playback = iota
	recording
	outputs
	inputs
	configuration
	panelCount
)

var titles = []string{"PLAYBACK", "RECORDING", "OUTPUT DEVICES", "INPUT DEVICES", "CONFIGURATION"}
var accent = lipgloss.Color("#81A1C1")
var dim = lipgloss.Color("#6B7486")
var bright = lipgloss.Color("#E5E9F0")
var cyan = lipgloss.Color("#88C0D0")
var green = lipgloss.Color("#A3BE8C")
var red = lipgloss.Color("#BF616A")

var titleStyle = lipgloss.NewStyle().Foreground(cyan).Bold(true)
var mutedStyle = lipgloss.NewStyle().Foreground(dim)
var selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#2E3440")).Background(cyan).Bold(true)

type refreshMsg struct {
	data AudioSnapshot
	err  error
}
type actionMsg struct {
	text string
	err  error
}
type tickMsg struct{}

type menuOption struct {
	label string
	args  []string
}
type model struct {
	data       AudioSnapshot
	focus      int
	cursor     [panelCount]int
	scroll     [panelCount]int
	width      int
	height     int
	status     string
	expanded   bool
	help       bool
	searching  bool
	search     string
	menuTitle  string
	menu       []menuOption
	menuCursor int
}

func refresh() tea.Cmd {
	return func() tea.Msg { s, err := loadSnapshot(); return refreshMsg{s, err} }
}
func runAction(args ...string) tea.Cmd {
	return func() tea.Msg { _, err := pactl(args...); return actionMsg{strings.Join(args, " "), err} }
}
func tick() tea.Cmd           { return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return tickMsg{} }) }
func (m model) Init() tea.Cmd { return tea.Batch(refresh(), tick()) }

func (m model) selectedID(p int) string {
	idx := m.cursor[p]
	switch p {
	case playback:
		if idx < len(m.data.SinkInputs) {
			return strconv.FormatUint(uint64(m.data.SinkInputs[idx].Index), 10)
		}
	case recording:
		if idx < len(m.data.SourceOutputs) {
			return strconv.FormatUint(uint64(m.data.SourceOutputs[idx].Index), 10)
		}
	case outputs:
		if idx < len(m.data.Sinks) {
			return m.data.Sinks[idx].Name
		}
	case inputs:
		if idx < len(m.data.Sources) {
			return m.data.Sources[idx].Name
		}
	case configuration:
		if idx < len(m.data.Cards) {
			return m.data.Cards[idx].Name
		}
	}
	return ""
}
func (m model) findID(p int, id string) int {
	if id == "" {
		return 0
	}
	n := m.count(p)
	for i := 0; i < n; i++ {
		m.cursor[p] = i
		if m.selectedID(p) == id {
			return i
		}
	}
	return 0
}
func (m model) count(p int) int {
	switch p {
	case playback:
		return len(m.data.SinkInputs)
	case recording:
		return len(m.data.SourceOutputs)
	case outputs:
		return len(m.data.Sinks)
	case inputs:
		return len(m.data.Sources)
	case configuration:
		return len(m.data.Cards)
	}
	return 0
}
func (m *model) changeFocus(p int) {
	m.focus = (p + panelCount) % panelCount
	m.search = ""
	m.searching = false
}

// Positions match the grid in View: playback | outputs, recording | inputs.
// Configuration is a separate page, reached via Tab or 5, not an adjacent widget.
var widgetPositions = [configuration]struct{ x, y int }{
	playback: {0, 0}, recording: {0, 1}, outputs: {1, 0}, inputs: {1, 1},
}

func (m *model) focusDirection(dx, dy int) {
	if m.focus >= configuration || (dx == 0 && dy == 0) {
		return
	}
	from := widgetPositions[m.focus]
	closest, distance := -1, int(^uint(0)>>1)
	for p, pos := range widgetPositions {
		x, y := pos.x-from.x, pos.y-from.y
		if (dx != 0 && y == 0 && x*dx > 0 && x*dx < distance) ||
			(dy != 0 && x == 0 && y*dy > 0 && y*dy < distance) {
			closest = p
			if dx != 0 {
				distance = x * dx
			} else {
				distance = y * dy
			}
		}
	}
	if closest >= 0 {
		m.changeFocus(closest)
	}
}

func (m *model) move(delta int) {
	if m.count(m.focus) == 0 {
		return
	}
	m.cursor[m.focus] = max(0, min(m.count(m.focus)-1, m.cursor[m.focus]+delta))
}
func (m *model) filterMove(forward bool) {
	if m.search == "" {
		return
	}
	n := m.count(m.focus)
	if n == 0 {
		return
	}
	start := m.cursor[m.focus]
	for step := 1; step <= n; step++ {
		i := (start + step) % n
		if !forward {
			i = (start - step + n*2) % n
		}
		if strings.Contains(strings.ToLower(m.rowName(m.focus, i)), strings.ToLower(m.search)) {
			m.cursor[m.focus] = i
			return
		}
	}
}
func (m model) rowName(p, i int) string {
	switch p {
	case playback:
		return m.data.SinkInputs[i].DisplayName
	case recording:
		return m.data.SourceOutputs[i].DisplayName
	case outputs:
		return m.data.Sinks[i].DisplayName
	case inputs:
		return m.data.Sources[i].DisplayName
	case configuration:
		return m.data.Cards[i].DisplayName
	}
	return ""
}
func (m model) volumeAction(change string) tea.Cmd {
	id := m.selectedID(m.focus)
	if id == "" || m.focus == configuration {
		return nil
	}
	verb := []string{"set-sink-input-volume", "set-source-output-volume", "set-sink-volume", "set-source-volume"}[m.focus]
	return runAction(verb, id, change)
}
func (m model) muteAction() tea.Cmd {
	id := m.selectedID(m.focus)
	if id == "" || m.focus == configuration {
		return nil
	}
	verb := []string{"set-sink-input-mute", "set-source-output-mute", "set-sink-mute", "set-source-mute"}[m.focus]
	return runAction(verb, id, "toggle")
}
func (m model) defaultAction() tea.Cmd {
	id := m.selectedID(m.focus)
	if id == "" {
		return nil
	}
	if m.focus == outputs {
		return runAction("set-default-sink", id)
	}
	if m.focus == inputs {
		return runAction("set-default-source", id)
	}
	return nil
}
func (m *model) openMenu() {
	id := m.selectedID(m.focus)
	if id == "" {
		return
	}
	m.menuCursor = 0
	m.menu = nil
	switch m.focus {
	case playback:
		m.menuTitle = "MOVE PLAYBACK TO OUTPUT"
		for _, s := range m.data.Sinks {
			m.menu = append(m.menu, menuOption{s.DisplayName, []string{"move-sink-input", id, s.Name}})
		}
	case recording:
		m.menuTitle = "MOVE RECORDING TO INPUT"
		for _, s := range m.data.Sources {
			m.menu = append(m.menu, menuOption{s.DisplayName, []string{"move-source-output", id, s.Name}})
		}
	case outputs:
		m.menuTitle = "OUTPUT PORT"
		for _, p := range m.data.Sinks[m.cursor[outputs]].Ports {
			m.menu = append(m.menu, menuOption{p.Description + " (" + p.Availability + ")", []string{"set-sink-port", id, p.Name}})
		}
	case inputs:
		m.menuTitle = "INPUT PORT"
		for _, p := range m.data.Sources[m.cursor[inputs]].Ports {
			m.menu = append(m.menu, menuOption{p.Description + " (" + p.Availability + ")", []string{"set-source-port", id, p.Name}})
		}
	case configuration:
		m.menuTitle = "CARD PROFILE"
		card := m.data.Cards[m.cursor[configuration]]
		keys := make([]string, 0, len(card.Profiles))
		for key := range card.Profiles {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			p := card.Profiles[key]
			mark := "  "
			if key == card.ActiveProfile {
				mark = "* "
			}
			if !p.Available {
				mark += "(unavailable) "
			}
			m.menu = append(m.menu, menuOption{mark + p.Description, []string{"set-card-profile", id, key}})
		}
	}
	if len(m.menu) == 0 {
		m.status = "No options available"
	}
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		return m, tea.Batch(refresh(), tick())
	case refreshMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			var ids [panelCount]string
			for p := range ids {
				ids[p] = m.selectedID(p)
			}
			m.data = msg.data
			for p := range ids {
				m.cursor[p] = m.findID(p, ids[p])
			}
		}
	case actionMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.status = "OK: " + msg.text
			return m, refresh()
		}
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if m.help {
			m.help = false
			return m, nil
		}
		if len(m.menu) > 0 {
			switch key {
			case "esc":
				m.menu = nil
			case "up", "k":
				m.menuCursor = max(0, m.menuCursor-1)
			case "down", "j":
				m.menuCursor = min(len(m.menu)-1, m.menuCursor+1)
			case "enter":
				args := m.menu[m.menuCursor].args
				m.menu = nil
				return m, runAction(args...)
			}
			return m, nil
		}
		if m.searching {
			switch key {
			case "esc", "enter":
				m.searching = false
			case "backspace", "ctrl+h":
				if len(m.search) > 0 {
					m.search = string([]rune(m.search)[:len([]rune(m.search))-1])
					m.filterMove(true)
				}
			default:
				if len(msg.Runes) > 0 {
					m.search += string(msg.Runes)
					m.filterMove(true)
				}
			}
			return m, nil
		}
		switch key {
		case "q":
			return m, tea.Quit
		case "?":
			m.help = true
		case "esc":
			m.expanded = false
			m.search = ""
		case "e":
			m.expanded = !m.expanded
		case "tab":
			m.changeFocus(m.focus + 1)
		case "shift+tab":
			m.changeFocus(m.focus - 1)
		case "1", "2", "3", "4", "5":
			m.changeFocus(int(key[0] - '1'))
		case "ctrl+up", "shift+up", "K", "W":
			m.focusDirection(0, -1)
		case "ctrl+down", "shift+down", "J", "S":
			m.focusDirection(0, 1)
		case "ctrl+left", "shift+left", "H", "A":
			m.focusDirection(-1, 0)
		case "ctrl+right", "shift+right", "L", "D":
			m.focusDirection(1, 0)
		case "up", "k":
			m.move(-1)
		case "down", "j":
			m.move(1)
		case "home", "g":
			m.cursor[m.focus] = 0
		case "end", "G":
			m.cursor[m.focus] = max(0, m.count(m.focus)-1)
		case "pgup":
			m.move(-max(1, m.height/2))
		case "pgdown":
			m.move(max(1, m.height/2))
		case "left", "h", "-":
			return m, m.volumeAction("-5%")
		case "right", "l", "+", "=":
			return m, m.volumeAction("+5%")
		case "m":
			return m, m.muteAction()
		case "d":
			return m, m.defaultAction()
		case "enter", "p":
			m.openMenu()
		case "/":
			m.searching = true
			m.search = ""
		case "n":
			m.filterMove(true)
		case "N":
			m.filterMove(false)
		case "r":
			return m, refresh()
		}
	}
	return m, nil
}

func truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	return ansi.Truncate(s, maxWidth, "…")
}
func volumeBar(percent, width int) string {
	if width < 1 {
		return ""
	}
	filled := min(width, max(0, percent*width/100))
	return lipgloss.NewStyle().Foreground(green).Render(strings.Repeat("━", filled)) + mutedStyle.Render(strings.Repeat("─", width-filled))
}
func (m model) row(p, i, width int) string {
	name := m.rowName(p, i)
	var vol map[string]ChannelVolume
	var muted bool
	var suffix string
	switch p {
	case playback:
		s := m.data.SinkInputs[i]
		vol, muted = s.Volume, s.Mute
		for _, dest := range m.data.Sinks {
			if dest.Index == s.Sink {
				suffix = " → " + dest.DisplayName
				break
			}
		}
	case recording:
		s := m.data.SourceOutputs[i]
		vol, muted = s.Volume, s.Mute
		for _, dest := range m.data.Sources {
			if dest.Index == s.Source {
				suffix = " ← " + dest.DisplayName
				break
			}
		}
	case outputs:
		s := m.data.Sinks[i]
		vol, muted = s.Volume, s.Mute
		if s.Name == m.data.DefaultSink {
			suffix = " ★ DEFAULT"
		} else if s.ActivePort != "" {
			suffix = " · " + s.ActivePort
		}
	case inputs:
		s := m.data.Sources[i]
		vol, muted = s.Volume, s.Mute
		if s.Name == m.data.DefaultSource {
			suffix = " ★ DEFAULT"
		} else if s.IsMonitor {
			suffix = " · Monitor"
		} else if s.ActivePort != "" {
			suffix = " · " + s.ActivePort
		}
	case configuration:
		s := m.data.Cards[i]
		suffix = " · " + s.ActiveProfile
	}
	if width < 10 {
		return truncate(name, width)
	}
	if p == configuration {
		return truncate(name+suffix, width)
	}
	v, pct, _ := getAvgVolume(vol)
	indicator := " "
	if muted {
		indicator = lipgloss.NewStyle().Foreground(red).Render("M")
	}
	barWidth := 10
	if width < 42 {
		barWidth = 5
	}
	available := max(1, width-barWidth-7-lipgloss.Width(pct))
	label := truncate(name+suffix, available)
	label = lipgloss.NewStyle().Width(available).Render(label)
	return fmt.Sprintf("%s %s %s %s", label, volumeBar(v, barWidth), pct, indicator)
}
func (m model) panel(p, width, height int) string {
	if width < 4 {
		width = 4
	}
	if height < 3 {
		height = 3
	}
	innerW := width - 4
	innerH := height - 2
	borderColor := dim
	if m.focus == p {
		borderColor = cyan
	}
	border := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(borderColor).Width(width-2).Height(height-2).Padding(0, 1)
	label := fmt.Sprintf(" %d %s ", p+1, titles[p])
	if m.focus == p {
		label = titleStyle.Render(label)
	} else {
		label = mutedStyle.Render(label)
	}
	n := m.count(p)
	if n == 0 {
		return border.Render(label + "\n" + mutedStyle.Render("(none)"))
	}
	cursor := m.cursor[p]
	scroll := m.scroll[p]
	// One line is reserved for the panel title.
	visible := max(1, innerH-1)
	if cursor < scroll {
		scroll = cursor
	}
	if cursor >= scroll+visible {
		scroll = cursor - visible + 1
	}
	if scroll > max(0, n-visible) {
		scroll = max(0, n-visible)
	}
	var lines []string
	lines = append(lines, truncate(label, innerW))
	for i := scroll; i < n && i < scroll+visible; i++ {
		row := m.row(p, i, innerW)
		if i == cursor && m.focus == p {
			// Re-render selected row as a solid highlighted line.
			row = selectedStyle.Width(innerW).Render(truncate(m.plainRow(p, i, innerW), innerW))
		}
		lines = append(lines, row)
	}
	return border.Render(strings.Join(lines, "\n"))
}

// plainRow mirrors the content of a selected row without ANSI escapes, so highlight covers the whole line.
func (m model) plainRow(p, i, width int) string {
	name := m.rowName(p, i)
	if p == configuration {
		return name + " · " + m.data.Cards[i].ActiveProfile
	}
	var vol map[string]ChannelVolume
	var muted bool
	switch p {
	case playback:
		vol, muted = m.data.SinkInputs[i].Volume, m.data.SinkInputs[i].Mute
	case recording:
		vol, muted = m.data.SourceOutputs[i].Volume, m.data.SourceOutputs[i].Mute
	case outputs:
		s := m.data.Sinks[i]
		vol, muted = s.Volume, s.Mute
		if s.Name == m.data.DefaultSink {
			name += " ★"
		}
	case inputs:
		s := m.data.Sources[i]
		vol, muted = s.Volume, s.Mute
		if s.Name == m.data.DefaultSource {
			name += " ★"
		}
		if s.IsMonitor {
			name += " · Monitor"
		}
	}
	_, pct, _ := getAvgVolume(vol)
	state := ""
	if muted {
		state = " MUTED"
	}
	return fmt.Sprintf("› %s  %s%s", truncate(name, max(1, width-lipgloss.Width(pct)-10)), pct, state)
}
func (m model) View() string {
	w, h := m.width, m.height
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 32
	}
	if w < 20 || h < 9 {
		return "aiyou: terminal too small (need 20x9)"
	}
	header := titleStyle.Render(" ◉ AIYOU ") + mutedStyle.Render("│") + "  PulseAudio / PipeWire  " + mutedStyle.Render("│") + "  " + fmt.Sprintf("OUT %d  IN %d  PLAY %d  REC %d", len(m.data.Sinks), len(m.data.Sources), len(m.data.SinkInputs), len(m.data.SourceOutputs))
	header = truncate(header, w)
	var body string
	availableH := max(3, h-4)
	if m.help {
		body = m.panelText("KEYBOARD HELP", []string{
			"Tab / Shift-Tab / 1-5   Switch widget", "Ctrl+arrows / H J K L / W A S D  Focus by position",
			"↑ ↓ / j k             Select item", "← → / h l / - +       Volume ±5%",
			"m  Mute/unmute         d  Set default input/output", "Enter / p   Route stream, choose port or profile",
			"e  Expand widget       Esc  Close / collapse", "/  Find item           n / N  Next / previous match",
			"r  Refresh             q / Ctrl-C  Quit", "", "Press any key to close",
		}, w, availableH)
	} else if len(m.menu) > 0 {
		var rows []string
		visible := max(1, availableH-5)
		start := max(0, m.menuCursor-visible+1)
		start = min(start, max(0, len(m.menu)-visible))
		for i := start; i < len(m.menu) && i < start+visible; i++ {
			line := truncate(m.menu[i].label, w-6)
			if i == m.menuCursor {
				line = selectedStyle.Render("› " + line)
			} else {
				line = "  " + line
			}
			rows = append(rows, line)
		}
		rows = append(rows, mutedStyle.Render(fmt.Sprintf("%d-%d/%d  ↑↓ choose  Enter apply  Esc cancel", start+1, min(len(m.menu), start+visible), len(m.menu))))
		body = m.panelText(m.menuTitle, rows, w, availableH)
	} else if m.expanded || m.focus == configuration || w < 70 || h < 20 {
		body = m.panel(m.focus, w, availableH)
	} else {
		leftW := (w - 1) / 2
		rightW := w - leftW - 1
		topH := availableH / 2
		bottomH := availableH - topH
		top := lipgloss.JoinHorizontal(lipgloss.Top, m.panel(playback, leftW, topH), " ", m.panel(outputs, rightW, topH))
		bottom := lipgloss.JoinHorizontal(lipgloss.Top, m.panel(recording, leftW, bottomH), " ", m.panel(inputs, rightW, bottomH))
		body = lipgloss.JoinVertical(lipgloss.Left, top, bottom)
	}
	status := m.status
	if m.searching {
		status = "/" + m.search + "_"
	} else if m.search != "" {
		status = "/" + m.search
	}
	if status == "" {
		status = "Ready"
	}
	footer := mutedStyle.Render(truncate("[Tab] widget  [↑↓] select  [←→] volume  [m] mute  [d] default  [Enter] options  [e] expand  [?] help  [q] quit", w))
	return lipgloss.JoinVertical(lipgloss.Left, header, body, truncate(status, w), footer)
}
func (m model) panelText(title string, rows []string, width, height int) string {
	st := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Width(width-2).Height(height-2).Padding(0, 1)
	var lines []string
	lines = append(lines, titleStyle.Render(" "+title+" "))
	visible := max(0, height-3)
	for i, r := range rows {
		if i >= visible {
			break
		}
		lines = append(lines, truncate(r, width-4))
	}
	if len(rows) > visible && visible > 0 {
		lines[len(lines)-1] = mutedStyle.Render(truncate("… more (enlarge terminal)", width-4))
	}
	return st.Render(strings.Join(lines, "\n"))
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
