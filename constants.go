package main

import "github.com/charmbracelet/lipgloss"

const (
	YYYYMMDD            = "2006-01-02"
	MMDDYYYY            = "01/02/2006"
	HHMMSS24h           = "15:04:05"
	HHMM24h             = "15:04"
	HHMMSS12h           = "3:04:05 PM"
	HHMM12h             = "3:04 PM"
	MMDDYYYYHHMM24h     = "01/02/2006 15:04"
	TextDate            = "January 2, 2006"
	TextDateWithWeekday = "Monday, January 2, 2006"
	AbbreviatedTextDate = "Mon Jan 2"
)

var (
	textInputPlaceholderStyle = lipgloss.NewStyle().Faint(true)
	titleStyle                = lipgloss.NewStyle().AlignHorizontal(lipgloss.Center)
	dateStyle                 = lipgloss.NewStyle().Width(11)
	timeStyle                 = lipgloss.NewStyle().Width(6)
)
