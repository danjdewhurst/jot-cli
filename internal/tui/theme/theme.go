package theme

import (
	catppuccin "github.com/catppuccin/go"
	"github.com/charmbracelet/lipgloss"
)

var f = catppuccin.Frappe

// Colours — Frappé palette as lipgloss.Color values.
var (
	Mantle   = lipgloss.Color(f.Mantle().Hex)
	Surface0 = lipgloss.Color(f.Surface0().Hex)
	Surface1 = lipgloss.Color(f.Surface1().Hex)
	Surface2 = lipgloss.Color(f.Surface2().Hex)
	Text     = lipgloss.Color(f.Text().Hex)
	Subtext0 = lipgloss.Color(f.Subtext0().Hex)
	Overlay0 = lipgloss.Color(f.Overlay0().Hex)
	Overlay1 = lipgloss.Color(f.Overlay1().Hex)
	Blue     = lipgloss.Color(f.Blue().Hex)
	Mauve    = lipgloss.Color(f.Mauve().Hex)
	Yellow   = lipgloss.Color(f.Yellow().Hex)
	Lavender = lipgloss.Color(f.Lavender().Hex)
	Green    = lipgloss.Color(f.Green().Hex)
	Crust    = lipgloss.Color(f.Crust().Hex)
)

// ── Status bar ──────────────────────────────────────────────────────────

var StatusBar = lipgloss.NewStyle().
	Background(Mantle).
	Foreground(Text).
	Padding(0, 1)

// ── List view ───────────────────────────────────────────────────────────

var (
	ListTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Blue).
			Padding(0, 0, 1, 0)

	ListSelected = lipgloss.NewStyle().
			Background(Surface0).
			Bold(true).
			Foreground(Text)

	ListCursor = lipgloss.NewStyle().
			Foreground(Lavender)

	ListTag = lipgloss.NewStyle().
		Foreground(Mauve)

	ListDim = lipgloss.NewStyle().
		Foreground(Overlay1)

	ListPin = lipgloss.NewStyle().
		Foreground(Yellow)
)

// ── Detail view ─────────────────────────────────────────────────────────

var (
	DetailTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Blue).
			PaddingLeft(1)

	DetailMeta = lipgloss.NewStyle().
			Foreground(Overlay1).
			PaddingLeft(1)

	DetailBody = lipgloss.NewStyle().
			Foreground(Text).
			PaddingLeft(1)

	TagPill = lipgloss.NewStyle().
		Background(Mauve).
		Foreground(Crust).
		Padding(0, 1)

	DetailPin = lipgloss.NewStyle().
			Foreground(Yellow)

	DetailRef = lipgloss.NewStyle().
			Foreground(Lavender).
			Bold(true)

	DetailBacklinkHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(Overlay1).
				PaddingLeft(1).
				PaddingTop(1)

	DetailBacklink = lipgloss.NewStyle().
			Foreground(Lavender).
			PaddingLeft(3)
)

// ── Compose view ────────────────────────────────────────────────────────

var (
	ComposeLabel = lipgloss.NewStyle().
			Bold(true).
			Foreground(Blue)
)

// ── Search view ─────────────────────────────────────────────────────────

var (
	SearchPrompt = lipgloss.NewStyle().
			Bold(true).
			Foreground(Blue)

	SearchDim = lipgloss.NewStyle().
			Foreground(Overlay1)

	SearchSelected = lipgloss.NewStyle().
			Background(Surface0).
			Bold(true).
			Foreground(Text)

	SearchCursor = lipgloss.NewStyle().
			Foreground(Lavender)
)

// ── Log renderer ───────────────────────────────────────────────────────

var (
	LogHash = lipgloss.NewStyle().
		Foreground(Yellow)

	LogTimestamp = lipgloss.NewStyle().
			Foreground(Overlay0)

	LogTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Text)

	LogTagKey = lipgloss.NewStyle().
			Foreground(Overlay1)

	LogTagValue = lipgloss.NewStyle().
			Foreground(Blue)
)

// ── Help view ───────────────────────────────────────────────────────────

var (
	HelpTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Blue)

	HelpDivider = lipgloss.NewStyle().
			Foreground(Surface2)

	HelpSection = lipgloss.NewStyle().
			Bold(true).
			Foreground(Lavender)

	HelpKey = lipgloss.NewStyle().
		Foreground(Green).
		Width(10)

	HelpDesc = lipgloss.NewStyle().
			Foreground(Subtext0)
)
