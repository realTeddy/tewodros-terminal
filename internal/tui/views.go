package tui

import (
	"charm.land/lipgloss/v2"
)

var (
	promptUserStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	promptHostStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	promptPathStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	promptAtStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	outputStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	errorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	dirStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	fileStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	welcomeStyle    = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("14")).
			Padding(1, 3)
)

func renderPrompt(cwd string) string {
	return promptUserStyle.Render("visitor") +
		promptAtStyle.Render("@") +
		promptHostStyle.Render("tewodros.me") +
		promptAtStyle.Render(":") +
		promptPathStyle.Render(cwd) +
		promptAtStyle.Render("$ ")
}

func renderWelcome() string {
	return welcomeStyle.Render(
		"tewodros.me — terminal portfolio\n\n" +
			"Welcome. Type 'help' to begin.",
	)
}
