package ui

import (
	"fmt"
	"strings"

	"github.com/Mrhunderb/aiyou/internal/audio"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

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
	var vol map[string]audio.ChannelVolume
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
	v, pct, _ := audio.AverageVolume(vol)
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
	var vol map[string]audio.ChannelVolume
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
	_, pct, _ := audio.AverageVolume(vol)
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
	} else if m.expanded || w < 70 || h < 20 {
		body = m.panel(m.focus, w, availableH)
	} else {
		leftW := w / 2
		rightW := w - leftW
		leftTop := availableH / 2
		rightTop := availableH / 3
		rightMiddle := availableH / 3
		left := lipgloss.JoinVertical(lipgloss.Left,
			m.panel(outputs, leftW, leftTop),
			m.panel(inputs, leftW, availableH-leftTop))
		right := lipgloss.JoinVertical(lipgloss.Left,
			m.panel(playback, rightW, rightTop),
			m.panel(recording, rightW, rightMiddle),
			m.panel(configuration, rightW, availableH-rightTop-rightMiddle))
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
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
