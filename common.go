package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/api/calendar/v3"
)

const (
	googleBlue = lipgloss.Color("#4285F4")
)

const (
	HHMM24h                        = "15:04"
	KitchenWithSpace               = "3:04 PM"
	HH_MM_XM                       = "03:04 PM"
	YYYYMMDD                       = "2006-01-02"
	AbbreviatedTextDate            = "Jan 2 2006"
	AbbreviatedTextDateWithWeekday = "Mon Jan 2"
	TextDateWithWeekday            = "Monday, January 2, 2006"
	AbbreviatedTextDate24h         = "Jan 2 2006 15:04"
	AbbreviatedTextDate12h         = "Jan 2 2006 03:04 PM"
)

const (
	summaryWidth = 40
	monthWidth   = 3 // Jan
	dayWidth     = 2 // 02
	yearWidth    = 4 // 2006
	timeWidth    = len(HH_MM_XM)
)

var (
	textInputPlaceholderStyle = lipgloss.NewStyle().Faint(true)
	textInputBaseStyle        = lipgloss.NewStyle().PaddingLeft(1).Border(lipgloss.RoundedBorder())
	dialogStyle               = lipgloss.NewStyle().
					Padding(1).
					Border(lipgloss.RoundedBorder()).
					Align(lipgloss.Center, lipgloss.Center)
)

func checkbox(label string, checked bool) string {
	if checked {
		return "[X] " + label
	} else {
		return "[ ] " + label
	}
}

func isAllDay(event *calendar.Event) bool {
	return event.Start.Date != ""
}

func newTextInput(charLimit int) textinput.Model {
	input := textinput.New()
	input.CharLimit = charLimit
	input.PlaceholderStyle = textInputPlaceholderStyle
	input.Prompt = ""
	return input
}

func focusNext(inputs []textinput.Model, focusIndex int) int {
	newIndex := (focusIndex + 1) % len(inputs)
	refocus(inputs, newIndex)
	return newIndex
}

func focusPrev(inputs []textinput.Model, focusIndex int) int {
	newIndex := focusIndex - 1
	if newIndex < 0 {
		newIndex = len(inputs) - 1
	}
	refocus(inputs, newIndex)
	return newIndex
}

func refocus(inputs []textinput.Model, focusIndex int) {
	for i := range inputs {
		inputs[i].Blur()
	}
	inputs[focusIndex].Focus()
}

func isEmpty(input textinput.Model) bool {
	return len(input.Value()) == 0
}

func autofillPlaceholder(input *textinput.Model) {
	input.SetValue(input.Placeholder)
}

func autofillEmptyInputs(inputs []textinput.Model) {
	for _, input := range inputs {
		if isEmpty(input) {
			autofillPlaceholder(&input)
		}
	}
}

func parseDateTimeInputs(month, day, year, t string) (time.Time, error) {
	text := fmt.Sprintf("%s %s %s %s", month, day, year, strings.ToUpper(t))
	var d time.Time
	var err error
	// TODO maybe use a package to guess more than just 3 layouts
	// Try parsing as three different layouts
	if d, err = time.ParseInLocation(AbbreviatedTextDate+" "+time.Kitchen, text, time.Local); err == nil {
		return d, nil
	}
	if d, err = time.ParseInLocation(AbbreviatedTextDate+" "+KitchenWithSpace, text, time.Local); err == nil {
		return d, nil
	}
	if d, err = time.ParseInLocation(AbbreviatedTextDate+" "+HHMM24h, text, time.Local); err == nil {
		return d, nil
	}
	return d, fmt.Errorf("Failed to parse datetime")
}

func toDateFields(date time.Time) (string, string, string) {
	month := date.Month().String()[:3]
	day := fmt.Sprintf("%02d", date.Day())
	year := fmt.Sprintf("%d", date.Year())
	return month, day, year
}

func autoformatDateTimeInputs(monthInput, dayInput, yearInput, timeInput *textinput.Model) {
	datetime, err := parseDateTimeInputs(monthInput.Value(), dayInput.Value(), yearInput.Value(), timeInput.Value())
	if err != nil {
		return
	}
	populateDateTimeInputs(datetime, monthInput, dayInput, yearInput, timeInput)
}

func populateDateTimeInputs(datetime time.Time, monthInput, dayInput, yearInput, timeInput *textinput.Model) {
	populateDateInputs(datetime, monthInput, dayInput, yearInput)
	populateTimeInput(datetime, timeInput)
}

func populateDateInputs(datetime time.Time, monthInput, dayInput, yearInput *textinput.Model) {
	monthText, dayText, yearText := toDateFields(datetime)
	monthInput.SetValue(monthText)
	dayInput.SetValue(dayText)
	yearInput.SetValue(yearText)
}

func populateTimeInput(datetime time.Time, timeInput *textinput.Model) {
	timeInput.SetValue(datetime.Format(HH_MM_XM))
}
