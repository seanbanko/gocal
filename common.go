package main

import "github.com/charmbracelet/lipgloss"

const (
	YYYYMMDD                       = "2006-01-02"
	MMDDYYYY                       = "01/02/2006"
	HHMMSS24h                      = "15:04:05"
	HHMM24h                        = "15:04"
	HHMMSS12h                      = "3:04:05 PM"
	HHMM12h                        = "3:04 PM"
	MMDDYYYYHHMM24h                = "01/02/2006 15:04"
	TextDate                       = "January 2, 2006"
	AbbreviatedTextDate            = "Jan 2 2006"
	TextDateWithWeekday            = "Monday, January 2, 2006"
	AbbreviatedTextDateWithWeekday = "Mon Jan 2"
	AbbreviatedTextDate24h         = "Jan 2 2006 15:04"
)

const (
	googleBlue = lipgloss.Color("#4285F4")
)

const (
	summaryWidth = 40
	monthWidth   = 3
	dayWidth     = 2
	yearWidth    = 4
	timeWidth    = 2
)

var (
	dialogStyle = lipgloss.NewStyle().
			Padding(1).
			Border(lipgloss.RoundedBorder()).
			Align(lipgloss.Center, lipgloss.Center)
	textInputPlaceholderStyle = lipgloss.NewStyle().Faint(true)
	textInputSummaryStyle = lipgloss.NewStyle().
				Width(summaryWidth + 2).
				PaddingLeft(1).
				Border(lipgloss.RoundedBorder())
	textInputMonthStyle = lipgloss.NewStyle().
				Width(monthWidth + 2).
				PaddingLeft(1).
				Border(lipgloss.RoundedBorder())
	textInputDayStyle = lipgloss.NewStyle().
				Width(dayWidth + 2).
				PaddingLeft(1).
				Border(lipgloss.RoundedBorder())
	textInputYearStyle = lipgloss.NewStyle().
				Width(yearWidth + 2).
				PaddingLeft(1).
				Border(lipgloss.RoundedBorder())
	textInputTimeStyle = lipgloss.NewStyle().
				Width(timeWidth + 2).
				PaddingLeft(1).
				Border(lipgloss.RoundedBorder())
)
