package views

import "strings"

type HelpView struct{}

func NewHelpView() HelpView { return HelpView{} }

func (h HelpView) View() string {
	var b strings.Builder
	b.WriteString("Key Bindings\n")
	b.WriteString("────────────\n\n")
	b.WriteString("Navigation\n")
	b.WriteString("  j/↓     Move down\n")
	b.WriteString("  k/↑     Move up\n")
	b.WriteString("  enter   Open note\n")
	b.WriteString("  esc     Go back\n")
	b.WriteString("  g/Home  Go to top\n")
	b.WriteString("  G/End   Go to bottom\n\n")
	b.WriteString("Actions\n")
	b.WriteString("  n       New note\n")
	b.WriteString("  e       Edit note\n")
	b.WriteString("  d       Archive note\n")
	b.WriteString("  p       Toggle pin\n")
	b.WriteString("  /       Search\n")
	b.WriteString("  ?       This help\n")
	b.WriteString("  q       Quit\n\n")
	b.WriteString("Compose\n")
	b.WriteString("  tab     Switch title/body\n")
	b.WriteString("  ctrl+s  Save\n")
	b.WriteString("  esc     Cancel\n")
	return b.String()
}
