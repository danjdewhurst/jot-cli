package views

import (
	"strings"

	"github.com/danjdewhurst/jot-cli/internal/tui/theme"
)

type HelpView struct{}

func NewHelpView() HelpView { return HelpView{} }

func (h HelpView) View() string {
	var b strings.Builder

	b.WriteString(theme.HelpTitle.Render("Key Bindings"))
	b.WriteString("\n")
	b.WriteString(theme.HelpDivider.Render("────────────────────────────"))
	b.WriteString("\n\n")

	section := func(name string, bindings [][2]string) {
		b.WriteString(theme.HelpSection.Render(name))
		b.WriteString("\n")
		for _, bind := range bindings {
			b.WriteString("  ")
			b.WriteString(theme.HelpKey.Render(bind[0]))
			b.WriteString(theme.HelpDesc.Render(bind[1]))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	section("Navigation", [][2]string{
		{"j/↓", "Move down"},
		{"k/↑", "Move up"},
		{"enter", "Open note"},
		{"esc", "Go back"},
		{"g/Home", "Go to top"},
		{"G/End", "Go to bottom"},
	})

	section("Actions", [][2]string{
		{"n", "New note"},
		{"e", "Edit note"},
		{"d", "Archive note"},
		{"p", "Toggle pin"},
		{"/", "Filter notes"},
		{"?", "This help"},
		{"q", "Quit"},
	})

	section("Compose", [][2]string{
		{"tab", "Switch title/body"},
		{"ctrl+s", "Save"},
		{"esc", "Cancel"},
	})

	return b.String()
}
