package main

import (
    tea "github.com/charmbracelet/bubbletea"
)

type model struct{}

func main() {
    tea.NewProgram(model{}) 
}

func (m model) Init() tea.Cmd {
    return nil
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
    return m, nil
}

func (m model) View() string {
    return ""
}
