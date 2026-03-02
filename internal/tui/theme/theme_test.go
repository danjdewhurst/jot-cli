package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestColours_AreDefined(t *testing.T) {
	// Verify that all expected colours are defined
	colours := map[string]lipgloss.Color{
		"Mantle":   Mantle,
		"Surface0": Surface0,
		"Surface1": Surface1,
		"Surface2": Surface2,
		"Text":     Text,
		"Subtext0": Subtext0,
		"Overlay0": Overlay0,
		"Overlay1": Overlay1,
		"Blue":     Blue,
		"Mauve":    Mauve,
		"Yellow":   Yellow,
		"Lavender": Lavender,
		"Green":    Green,
		"Crust":    Crust,
	}

	for name, colour := range colours {
		if colour == "" {
			t.Errorf("%s colour is empty", name)
		}
	}
}

func TestStyles_RenderWithoutPanic(t *testing.T) {
	// Verify that all styles can render without panicking
	tests := []struct {
		name  string
		style lipgloss.Style
	}{
		{"StatusBar", StatusBar},
		{"ListTitle", ListTitle},
		{"ListSelected", ListSelected},
		{"ListCursor", ListCursor},
		{"ListTag", ListTag},
		{"ListDim", ListDim},
		{"ListPin", ListPin},
		{"ListCheck", ListCheck},
		{"DetailTitle", DetailTitle},
		{"DetailMeta", DetailMeta},
		{"DetailBody", DetailBody},
		{"TagPill", TagPill},
		{"DetailPin", DetailPin},
		{"DetailRef", DetailRef},
		{"DetailBacklinkHeader", DetailBacklinkHeader},
		{"DetailBacklink", DetailBacklink},
		{"ComposeLabel", ComposeLabel},
		{"SearchPrompt", SearchPrompt},
		{"SearchDim", SearchDim},
		{"SearchSelected", SearchSelected},
		{"SearchCursor", SearchCursor},
		{"LogHash", LogHash},
		{"LogTimestamp", LogTimestamp},
		{"LogTitle", LogTitle},
		{"LogTagKey", LogTagKey},
		{"LogTagValue", LogTagValue},
		{"HelpTitle", HelpTitle},
		{"HelpDivider", HelpDivider},
		{"HelpSection", HelpSection},
		{"HelpKey", HelpKey},
		{"HelpDesc", HelpDesc},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			_ = tt.style.String()
			_ = tt.style.Render("test")
		})
	}
}

func TestStatusBarStyle(t *testing.T) {
	rendered := StatusBar.Render("status message")
	if rendered == "" {
		t.Error("StatusBar style produced empty output")
	}
	if rendered == "status message" {
		t.Error("StatusBar style appears to have no styling applied")
	}
}

func TestListStyles(t *testing.T) {
	t.Run("ListTitle", func(t *testing.T) {
		rendered := ListTitle.Render("Notes")
		if rendered == "" {
			t.Error("ListTitle produced empty output")
		}
	})

	t.Run("ListSelected", func(t *testing.T) {
		rendered := ListSelected.Render("selected item")
		if rendered == "" {
			t.Error("ListSelected produced empty output")
		}
	})

	t.Run("ListPin", func(t *testing.T) {
		rendered := ListPin.Render("")
		// Empty string rendered might be empty, but that's ok for pin icon
		_ = rendered
	})
}

func TestDetailStyles(t *testing.T) {
	t.Run("DetailTitle", func(t *testing.T) {
		rendered := DetailTitle.Render("Note Title")
		if rendered == "" {
			t.Error("DetailTitle produced empty output")
		}
	})

	t.Run("DetailBody", func(t *testing.T) {
		rendered := DetailBody.Render("Body content")
		if rendered == "" {
			t.Error("DetailBody produced empty output")
		}
	})

	t.Run("TagPill", func(t *testing.T) {
		rendered := TagPill.Render("folder:work")
		if rendered == "" {
			t.Error("TagPill produced empty output")
		}
	})

	t.Run("DetailRef", func(t *testing.T) {
		rendered := DetailRef.Render("@abc123")
		if rendered == "" {
			t.Error("DetailRef produced empty output")
		}
	})
}

func TestComposeStyles(t *testing.T) {
	rendered := ComposeLabel.Render("Title:")
	if rendered == "" {
		t.Error("ComposeLabel produced empty output")
	}
}

func TestSearchStyles(t *testing.T) {
	rendered := SearchPrompt.Render("Search:")
	if rendered == "" {
		t.Error("SearchPrompt produced empty output")
	}
}

func TestLogStyles(t *testing.T) {
	tests := []struct {
		name  string
		style lipgloss.Style
		text  string
	}{
		{"LogHash", LogHash, "abc123"},
		{"LogTimestamp", LogTimestamp, "2024-01-01"},
		{"LogTitle", LogTitle, "Commit message"},
		{"LogTagKey", LogTagKey, "Author"},
		{"LogTagValue", LogTagValue, "John"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := tt.style.Render(tt.text)
			if rendered == "" {
				t.Errorf("%s produced empty output", tt.name)
			}
		})
	}
}

func TestHelpStyles(t *testing.T) {
	tests := []struct {
		name  string
		style lipgloss.Style
		text  string
	}{
		{"HelpTitle", HelpTitle, "Help"},
		{"HelpDivider", HelpDivider, "---"},
		{"HelpSection", HelpSection, "Navigation"},
		{"HelpKey", HelpKey, "j"},
		{"HelpDesc", HelpDesc, "move down"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := tt.style.Render(tt.text)
			if rendered == "" {
				t.Errorf("%s produced empty output", tt.name)
			}
		})
	}
}
