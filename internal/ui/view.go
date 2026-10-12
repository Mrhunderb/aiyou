package ui

import (
	"fmt"
	"strings"

	"github.com/Mrhunderb/ayou/internal/audio"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	return ansi.Truncate(s, maxWidth, "…")
}
func (m model) volumeBar(percent, width int) string {
	if width < 1 {
		return ""
	}
	filled := min(width, max(0, percent*width/100))
	return lipgloss.NewStyle().Foreground(m.theme.volume).Render(strings.Repeat("━", filled)) + m.theme.mutedStyle.Render(strings.Repeat("─", width-filled))
}

// A fixed leading cell keeps the default-device marker visible even when
// long device names are truncated or the row is selected.
func (m model) defaultMarker(p, i int) string {
	switch p {
	case outputs:
		if m.data.DefaultSink != "" && m.data.Sinks[i].Name == m.data.DefaultSink {
			return "★ "
		}
		return "  "
	case inputs:
		if m.data.DefaultSource != "" && m.data.Sources[i].Name == m.data.DefaultSource {
			return "★ "
		}
		return "  "
	}
	return ""
}
func (m model) row(p, i, width int) string {
	name := m.rowName(p, i)
	marker := m.defaultMarker(p, i)
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
		if s.ActivePort != "" {
			suffix = " · " + s.ActivePort
		}
	case inputs:
		s := m.data.Sources[i]
		vol, muted = s.Volume, s.Mute
		if s.IsMonitor {
			suffix = " · Monitor"
		} else if s.ActivePort != "" {
			suffix = " · " + s.ActivePort
		}
	case configuration:
		s := m.data.Cards[i]
		suffix = " · " + s.ActiveProfile
	}
	if width < 10 {
		return truncate(marker+name, width)
	}
	if p == configuration {
		return truncate(name+suffix, width)
	}
	v, pct, _ := audio.AverageVolume(vol)
	indicator := " "
	if muted {
		indicator = lipgloss.NewStyle().Foreground(m.theme.muted).Render("M")
	}
	barWidth := m.config.Layout.VolumeBarWidth
	if width < m.config.Layout.CompactBarThreshold {
		barWidth = m.config.Layout.CompactBarWidth
	}
	// Even a user-selected wide bar must fit within a narrow panel.
	barWidth = min(barWidth, max(1, width-lipgloss.Width(pct)-lipgloss.Width(marker)-8))
	available := max(1, width-barWidth-7-lipgloss.Width(pct)-lipgloss.Width(marker))
	label := truncate(name+suffix, available)
	label = lipgloss.NewStyle().Width(available).Render(label)
	if strings.HasPrefix(marker, "★") {
		marker = lipgloss.NewStyle().Foreground(m.theme.volume).Render(marker)
	}
	return fmt.Sprintf("%s%s %s %s %s", marker, label, m.volumeBar(v, barWidth), pct, indicator)
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
	borderColor := m.theme.dim
	if m.focus == p {
		borderColor = m.theme.active
	}
	border := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(borderColor).Width(width-2).Height(height-2).Padding(0, 1)
	label := fmt.Sprintf(" %d %s ", p+1, titles[p])
	if m.focus == p {
		label = m.theme.titleStyle.Render(label)
	} else {
		label = m.theme.mutedStyle.Render(label)
	}
	matches := m.visibleIndices(p)
	if len(matches) == 0 {
		message := "(none)"
		if m.count(p) > 0 {
			message = "(no matches)"
		}
		return border.Render(label + "\n" + m.theme.mutedStyle.Render(message))
	}
	cursor := m.cursor[p]
	cursorRow := indexOf(matches, cursor)
	scroll := m.scroll[p]
	// Scroll and page size are measured in visible rows, not snapshot indices.
	pageSize := max(1, innerH-1)
	if cursorRow >= 0 && cursorRow < scroll {
		scroll = cursorRow
	}
	if cursorRow >= scroll+pageSize {
		scroll = cursorRow - pageSize + 1
	}
	if scroll > max(0, len(matches)-pageSize) {
		scroll = max(0, len(matches)-pageSize)
	}
	var lines []string
	lines = append(lines, truncate(label, innerW))
	for rowIndex := scroll; rowIndex < len(matches) && rowIndex < scroll+pageSize; rowIndex++ {
		i := matches[rowIndex]
		row := m.row(p, i, innerW)
		if i == cursor && m.focus == p {
			row = m.theme.selectedStyle.Width(innerW).Render(truncate(m.plainRow(p, i, innerW), innerW))
		}
		lines = append(lines, row)
	}
	return border.Render(strings.Join(lines, "\n"))
}

// plainRow mirrors the content of a selected row without ANSI escapes, so highlight covers the whole line.
func (m model) plainRow(p, i, width int) string {
	name := m.rowName(p, i)
	marker := m.defaultMarker(p, i)
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
	case inputs:
		s := m.data.Sources[i]
		vol, muted = s.Volume, s.Mute
		if s.IsMonitor {
			name += " · Monitor"
		}
	}
	_, pct, _ := audio.AverageVolume(vol)
	state := ""
	if muted {
		state = " MUTED"
	}
	return fmt.Sprintf("› %s%s  %s%s", marker, truncate(name, max(1, width-lipgloss.Width(marker)-lipgloss.Width(pct)-10)), pct, state)
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
		return "ayou: terminal too small (need 20x9)"
	}
	header := m.theme.titleStyle.Render(" ◉ AYOU ") + m.theme.mutedStyle.Render("│") + "  PulseAudio / PipeWire  " + m.theme.mutedStyle.Render("│") + "  " + fmt.Sprintf("OUT %d  IN %d  PLAY %d  REC %d", len(m.data.Sinks), len(m.data.Sources), len(m.data.SinkInputs), len(m.data.SourceOutputs))
	header = truncate(header, w)
	var body string
	availableH := max(3, h-4)
	if m.help {
		body = m.panelText("KEYBOARD HELP", []string{
			"Tab / Shift-Tab / 1-5   Switch widget", "Ctrl+arrows / H J K L / W A S D  Focus by position",
			"↑ ↓ / j k             Select item", fmt.Sprintf("← → / h l / - +       Volume ±%d%%", m.config.VolumeStep),
			"m  Mute/unmute         d  Set default input/output", "Enter / p   Route stream, choose port or profile",
			"e  Expand widget       Esc  Close / collapse", "/  Filter this widget  n / N  Next / previous match",
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
				line = m.theme.selectedStyle.Render("› " + line)
			} else {
				line = "  " + line
			}
			rows = append(rows, line)
		}
		rows = append(rows, m.theme.mutedStyle.Render(fmt.Sprintf("%d-%d/%d  ↑↓ choose  Enter apply  Esc cancel", start+1, min(len(m.menu), start+visible), len(m.menu))))
		body = m.panelText(m.menuTitle, rows, w, availableH)
	} else if m.expanded || w < m.config.Layout.DashboardMinWidth || h < m.config.Layout.DashboardMinHeight {
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
	footer := m.theme.mutedStyle.Render(truncate("[Tab] widget  [↑↓] select  [←→] volume  [m] mute  [d] default  [Enter] options  [e] expand  [?] help  [q] quit", w))
	return lipgloss.JoinVertical(lipgloss.Left, header, body, truncate(status, w), footer)
}
func (m model) panelText(title string, rows []string, width, height int) string {
	st := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(m.theme.accent).Width(width-2).Height(height-2).Padding(0, 1)
	var lines []string
	lines = append(lines, m.theme.titleStyle.Render(" "+title+" "))
	visible := max(0, height-3)
	for i, r := range rows {
		if i >= visible {
			break
		}
		lines = append(lines, truncate(r, width-4))
	}
	if len(rows) > visible && visible > 0 {
		lines[len(lines)-1] = m.theme.mutedStyle.Render(truncate("… more (enlarge terminal)", width-4))
	}
	return st.Render(strings.Join(lines, "\n"))
}
