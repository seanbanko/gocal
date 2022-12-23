package main

import (
	"github.com/charmbracelet/bubbles/key"
)

type keyMap struct {
	Next   key.Binding
	Prev   key.Binding
	Today  key.Binding
	Create key.Binding
	Help   key.Binding
	Quit   key.Binding
}

var DefaultKeyMap = keyMap{
	Next: key.NewBinding(
		key.WithKeys("n", "p"),
		key.WithHelp("n/j", "next period"),
	),
	Prev: key.NewBinding(
		key.WithKeys("p", "k"),
		key.WithHelp("p/k", "previous period"),
	),
	Today: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "today"),
	),
	Create: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "create event"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("ctrl+c/q", "quit"),
	),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Next, k.Help},
		{k.Prev, k.Quit},
		{k.Today},
		{k.Create},
	}
}

type keyMapCreate struct {
	Next key.Binding
	Prev   key.Binding
	Create key.Binding
	Cancel   key.Binding
	Quit   key.Binding
}

var CreateKeyMap = keyMapCreate{
	Next: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next field"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "previous field"),
	),
	Create: key.NewBinding(
		key.WithKeys("enter", "ctrl+s"),
		key.WithHelp("enter/ctrl+s", "create event"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("ctrl+c/q", "quit"),
	),
}

func (k keyMapCreate) ShortHelp() []key.Binding {
	return []key.Binding{k.Next, k.Prev, k.Create, k.Cancel, k.Quit}
}

func (k keyMapCreate) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Next, k.Cancel},
		{k.Prev, k.Quit},
		{k.Create},
	}
}
