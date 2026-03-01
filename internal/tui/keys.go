package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Quit     key.Binding
	Help     key.Binding
	Enter    key.Binding
	Back     key.Binding
	New      key.Binding
	Delete   key.Binding
	Search   key.Binding
	Edit     key.Binding
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
}

var keys = keyMap{
	Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
	Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	New:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new note")),
	Delete:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "archive")),
	Search:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	Edit:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
	Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	PageUp:   key.NewBinding(key.WithKeys("pgup", "ctrl+u"), key.WithHelp("pgup", "page up")),
	PageDown: key.NewBinding(key.WithKeys("pgdown", "ctrl+d"), key.WithHelp("pgdn", "page down")),
}
