package main

import "github.com/charmbracelet/lipgloss"

var (
	dialogStyle = lipgloss.NewStyle().
			Padding(1).
			Border(lipgloss.RoundedBorder()).
			AlignHorizontal(lipgloss.Center).
			AlignVertical(lipgloss.Center)
	textInputPlaceholderStyle = lipgloss.NewStyle().Faint(true)
	titleStyle                = lipgloss.NewStyle().AlignHorizontal(lipgloss.Center)
	dateStyle                 = lipgloss.NewStyle().Width(11)
	timeStyle                 = lipgloss.NewStyle().Width(6)
)
