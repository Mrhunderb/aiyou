package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		return m, tea.Batch(m.refresh(), tick())
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
			return m, m.refresh()
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
				action := m.menu[m.menuCursor].action
				m.menu = nil
				return m, m.runAction(action)
			}
			return m, nil
		}
		if m.searching {
			switch key {
			case "esc", "enter":
				m.searching = false
			case "up":
				m.move(-1)
			case "down":
				m.move(1)
			case "home":
				m.moveToBoundary(false)
			case "end":
				m.moveToBoundary(true)
			case "pgup":
				m.move(-max(1, m.height/2))
			case "pgdown":
				m.move(max(1, m.height/2))
			case "backspace", "ctrl+h":
				if len(m.search) > 0 {
					m.search = string([]rune(m.search)[:len([]rune(m.search))-1])
					m.normalizeSelection()
				}
			default:
				if len(msg.Runes) > 0 {
					m.search += string(msg.Runes)
					m.normalizeSelection()
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
			m.moveToBoundary(false)
		case "end", "G":
			m.moveToBoundary(true)
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
			return m, m.refresh()
		}
	}
	return m, nil
}
