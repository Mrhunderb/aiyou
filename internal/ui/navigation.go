package ui

import (
	"strconv"
	"strings"
)

// Cursors store indices in the unfiltered snapshot so actions always target
// the original object, regardless of the displayed row position.
func (m model) selectedID(p int) string {
	idx := m.cursor[p]
	if idx < 0 || idx >= m.count(p) || !m.matchesSearch(p, idx) {
		return ""
	}
	return m.rowID(p, idx)
}
func (m model) rowID(p, idx int) string {
	switch p {
	case playback:
		return strconv.FormatUint(uint64(m.data.SinkInputs[idx].Index), 10)
	case recording:
		return strconv.FormatUint(uint64(m.data.SourceOutputs[idx].Index), 10)
	case outputs:
		return m.data.Sinks[idx].Name
	case inputs:
		return m.data.Sources[idx].Name
	case configuration:
		return m.data.Cards[idx].Name
	}
	return ""
}
func (m model) findID(p int, id string) int {
	if id != "" {
		for i := 0; i < m.count(p); i++ {
			if m.rowID(p, i) == id && m.matchesSearch(p, i) {
				return i
			}
		}
	}
	visible := m.visibleIndices(p)
	if len(visible) > 0 {
		return visible[0]
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
	// Expansion is modal: selection and audio controls remain available,
	// but widget navigation is disabled until e or Esc collapses it.
	if m.expanded {
		return
	}
	m.focus = (p + panelCount) % panelCount
	m.search = ""
	m.searching = false
}

// Widget centers match the two left rows and three right rows in View.
// The y coordinates are twelfths of the available height (3/9 vs 2/6/10).
// Horizontal movement picks the closest vertical center in the other column;
// on a tie it prefers the lower widget (recording -> input devices).
var widgetPositions = [panelCount]struct{ x, y int }{
	outputs: {0, 3}, inputs: {0, 9},
	playback: {1, 2}, recording: {1, 6}, configuration: {1, 10},
}

func (m *model) focusDirection(dx, dy int) {
	if m.expanded || (dx == 0 && dy == 0) {
		return
	}
	from := widgetPositions[m.focus]
	closest, distance := -1, int(^uint(0)>>1)
	for p, pos := range widgetPositions {
		x, y := pos.x-from.x, pos.y-from.y
		if dx != 0 {
			if x*dx <= 0 {
				continue
			}
			d := max(y, -y)
			if d < distance || (d == distance && (closest < 0 || pos.y > widgetPositions[closest].y)) {
				closest, distance = p, d
			}
		} else {
			if x != 0 || y*dy <= 0 {
				continue
			}
			d := max(y, -y)
			if d < distance {
				closest, distance = p, d
			}
		}
	}
	if closest >= 0 {
		m.changeFocus(closest)
	}
}

func (m model) matchesSearch(p, i int) bool {
	return m.search == "" || p != m.focus ||
		strings.Contains(strings.ToLower(m.rowName(p, i)), strings.ToLower(m.search))
}

// visibleIndices maps a filtered row back to the original snapshot index.
// Only the focused widget is filtered; other widgets keep their own contents.
func (m model) visibleIndices(p int) []int {
	visible := make([]int, 0, m.count(p))
	for i := 0; i < m.count(p); i++ {
		if m.matchesSearch(p, i) {
			visible = append(visible, i)
		}
	}
	return visible
}
func indexOf(indices []int, original int) int {
	for i, idx := range indices {
		if idx == original {
			return i
		}
	}
	return -1
}
func (m *model) normalizeSelection() {
	visible := m.visibleIndices(m.focus)
	if len(visible) == 0 {
		m.cursor[m.focus] = 0
		return
	}
	if indexOf(visible, m.cursor[m.focus]) < 0 {
		m.cursor[m.focus] = visible[0]
	}
}
func (m *model) move(delta int) {
	visible := m.visibleIndices(m.focus)
	if len(visible) == 0 {
		return
	}
	pos := indexOf(visible, m.cursor[m.focus])
	if pos < 0 {
		pos = 0
	} else {
		pos = max(0, min(len(visible)-1, pos+delta))
	}
	m.cursor[m.focus] = visible[pos]
}
func (m *model) moveToBoundary(last bool) {
	visible := m.visibleIndices(m.focus)
	if len(visible) == 0 {
		return
	}
	if last {
		m.cursor[m.focus] = visible[len(visible)-1]
	} else {
		m.cursor[m.focus] = visible[0]
	}
}
func (m *model) filterMove(forward bool) {
	if m.search == "" {
		return
	}
	visible := m.visibleIndices(m.focus)
	if len(visible) == 0 {
		return
	}
	pos := indexOf(visible, m.cursor[m.focus])
	if pos < 0 {
		pos = 0
	} else if forward {
		pos = (pos + 1) % len(visible)
	} else {
		pos = (pos - 1 + len(visible)) % len(visible)
	}
	m.cursor[m.focus] = visible[pos]
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
