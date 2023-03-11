package main

import "github.com/charmbracelet/lipgloss"

var (
	dialogStyle = lipgloss.NewStyle().
			Padding(1).
			Border(lipgloss.RoundedBorder()).
			Align(lipgloss.Center, lipgloss.Center)
	textInputPlaceholderStyle = lipgloss.NewStyle().Faint(true)
	textInputSummaryStyle        = lipgloss.NewStyle().AlignHorizontal(lipgloss.Center)
	textInputDateStyle        = lipgloss.NewStyle().Width(12)
	textInputTimeStyle        = lipgloss.NewStyle().Width(6)
)
