package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/danjdewhurst/jot-cli/internal/model"
)

// Markdown renders notes as a human-readable Markdown document.
func Markdown(w io.Writer, notes []model.Note) error {
	for i, n := range notes {
		if i > 0 {
			if _, err := fmt.Fprint(w, "---\n\n"); err != nil {
				return err
			}
		}

		if n.Title != "" {
			if _, err := fmt.Fprintf(w, "# %s\n\n", n.Title); err != nil {
				return err
			}
		}

		if _, err := fmt.Fprintf(w, "- **ID:** %s\n", n.ID); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "- **Created:** %s\n", n.CreatedAt.Format("2006-01-02T15:04:05Z07:00")); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "- **Updated:** %s\n", n.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")); err != nil {
			return err
		}
		if n.Archived {
			if _, err := fmt.Fprint(w, "- **Archived:** true\n"); err != nil {
				return err
			}
		}
		if len(n.Tags) > 0 {
			tagStrs := make([]string, len(n.Tags))
			for j, t := range n.Tags {
				tagStrs[j] = t.String()
			}
			if _, err := fmt.Fprintf(w, "- **Tags:** %s\n", strings.Join(tagStrs, ", ")); err != nil {
				return err
			}
		}

		if n.Body != "" {
			if _, err := fmt.Fprintf(w, "\n%s\n\n", n.Body); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprint(w, "\n"); err != nil {
				return err
			}
		}
	}
	return nil
}
