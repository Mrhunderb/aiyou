package ui

import (
	"github.com/Mrhunderb/ayou/internal/config"
	"github.com/charmbracelet/lipgloss"
)

// Each model owns its theme; changing config does not mutate global styles.
type theme struct {
	accent        lipgloss.Color
	dim           lipgloss.Color
	active        lipgloss.Color
	volume        lipgloss.Color
	muted         lipgloss.Color
	titleStyle    lipgloss.Style
	mutedStyle    lipgloss.Style
	selectedStyle lipgloss.Style
}

func newTheme(c config.Theme) theme {
	return theme{
		accent: lipgloss.Color(c.Accent), dim: lipgloss.Color(c.Dim),
		active: lipgloss.Color(c.Active), volume: lipgloss.Color(c.Volume), muted: lipgloss.Color(c.Muted),
		titleStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color(c.Active)).Bold(true),
		mutedStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color(c.Dim)),
		selectedStyle: lipgloss.NewStyle().Foreground(lipgloss.Color(c.SelectionForeground)).Background(lipgloss.Color(c.SelectionBackground)).Bold(true),
	}
}
