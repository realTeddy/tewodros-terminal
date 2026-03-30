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
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	subtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
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
	return "\r\n" +
		titleStyle.Render("  tewodros.me") + subtitleStyle.Render(" — not a fake terminal. ") +
		promptHostStyle.Render("ssh tewodros.me") + subtitleStyle.Render(" if you don't believe me.") + "\r\n" +
		"\r\n" +
		"  Try " + promptUserStyle.Render("about") + ", " +
		promptUserStyle.Render("contact") + ", or " +
		promptUserStyle.Render("help") + " to see all commands.\r\n"
}
