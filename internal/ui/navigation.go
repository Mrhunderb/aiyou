package ui

import (
	"strconv"
	"strings"
)

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
