package ui

import "github.com/charmbracelet/lipgloss"

// Colors and styles are centralized so rendering code has no theme literals.
var accent = lipgloss.Color("#81A1C1")
var dim = lipgloss.Color("#6B7486")
var cyan = lipgloss.Color("#88C0D0")
var green = lipgloss.Color("#A3BE8C")
var red = lipgloss.Color("#BF616A")

var titleStyle = lipgloss.NewStyle().Foreground(cyan).Bold(true)
var mutedStyle = lipgloss.NewStyle().Foreground(dim)
var selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#2E3440")).Background(cyan).Bold(true)
