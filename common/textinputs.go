package common

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

func NewTextInput(charLimit int) textinput.Model {
	input := textinput.New()
	input.CharLimit = charLimit
	input.PlaceholderStyle = lipgloss.NewStyle().Faint(true)
	input.Prompt = " "
	return input
}

func Refocus(inputs []textinput.Model, focusIndex int) {
	for i := range inputs {
		inputs[i].Blur()
	}
	inputs[focusIndex].Focus()
}

func IsEmpty(input textinput.Model) bool {
	return len(input.Value()) == 0
}

func AutofillPlaceholder(input *textinput.Model) {
	input.SetValue(input.Placeholder)
}

func AutofillEmptyInputs(inputs []textinput.Model) {
	for _, input := range inputs {
		if IsEmpty(input) {
			AutofillPlaceholder(&input)
		}
	}
}
