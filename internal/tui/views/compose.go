package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/danjdewhurst/jot-cli/internal/model"
)

var composeLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

type ComposeView struct {
	noteID    string
	titleIn   textinput.Model
	bodyIn    textarea.Model
	focusBody bool
	width     int
	height    int
}

func NewComposeView() ComposeView {
	ti := textinput.New()
	ti.Placeholder = "Title"
	ti.CharLimit = 200
	ti.Focus()

	ta := textarea.New()
	ta.Placeholder = "Write your note…"
	ta.ShowLineNumbers = false

	return ComposeView{
		titleIn: ti,
		bodyIn:  ta,
	}
}

func (c *ComposeView) Reset() {
	c.noteID = ""
	c.titleIn.Reset()
	c.bodyIn.Reset()
	c.focusBody = false
	c.titleIn.Focus()
	c.bodyIn.Blur()
}

func (c *ComposeView) SetNote(n model.Note) {
	c.noteID = n.ID
	c.titleIn.SetValue(n.Title)
	c.bodyIn.SetValue(n.Body)
	c.focusBody = false
	c.titleIn.Focus()
	c.bodyIn.Blur()
}

func (c *ComposeView) NoteID() string {
	return c.noteID
}

func (c *ComposeView) Content() (string, string) {
	return strings.TrimSpace(c.titleIn.Value()), strings.TrimSpace(c.bodyIn.Value())
}

func (c *ComposeView) SetSize(w, h int) {
	c.width = w
	c.height = h
	c.titleIn.Width = w - 4
	c.bodyIn.SetWidth(w - 2)
	c.bodyIn.SetHeight(h - 6)
}

func (c *ComposeView) Update(msg tea.Msg) {
	if kmsg, ok := msg.(tea.KeyMsg); ok {
		if kmsg.String() == "tab" {
			c.focusBody = !c.focusBody
			if c.focusBody {
				c.titleIn.Blur()
				c.bodyIn.Focus()
			} else {
				c.bodyIn.Blur()
				c.titleIn.Focus()
			}
			return
		}
	}

	if c.focusBody {
		var cmd tea.Cmd
		c.bodyIn, cmd = c.bodyIn.Update(msg)
		_ = cmd
	} else {
		var cmd tea.Cmd
		c.titleIn, cmd = c.titleIn.Update(msg)
		_ = cmd
	}
}

func (c ComposeView) View() string {
	var b strings.Builder

	label := "New note"
	if c.noteID != "" {
		label = "Edit note"
	}
	b.WriteString(composeLabelStyle.Render(label))
	b.WriteString("\n\n")
	b.WriteString(c.titleIn.View())
	b.WriteString("\n\n")
	b.WriteString(c.bodyIn.View())

	return b.String()
}
