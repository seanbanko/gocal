package common

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
)

type Component interface {
	tea.Model
	help.KeyMap
}
