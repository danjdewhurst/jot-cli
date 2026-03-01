package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/danjdewhurst/jot-cli/internal/context"
	"github.com/danjdewhurst/jot-cli/internal/editor"
	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Create a new note",
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		body, _ := cmd.Flags().GetString("message")
		noContext, _ := cmd.Flags().GetBool("no-context")

		// Read from stdin if piped
		if body == "" && !term.IsTerminal(int(os.Stdin.Fd())) {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("reading stdin: %w", err)
			}
			body = strings.TrimSpace(string(data))
		}

		// Open editor if no body provided
		if body == "" {
			initial := ""
			if title != "" {
				initial = "# " + title + "\n\n"
			}
			edited, err := editor.Edit(initial)
			if err != nil {
				return fmt.Errorf("editor: %w", err)
			}
			body = strings.TrimSpace(edited)
			if body == "" {
				return fmt.Errorf("empty note, aborting")
			}

			// Extract title from first markdown heading if not set
			if title == "" {
				title, body = extractTitle(body)
			}
		}

		var tags []model.Tag
		if !noContext {
			tags = context.AutoTags()
		}

		// Add user-specified tags
		tagStrs, _ := cmd.Flags().GetStringSlice("tag")
		for _, ts := range tagStrs {
			t, err := model.ParseTag(ts)
			if err != nil {
				return err
			}
			tags = append(tags, t)
		}

		note, err := db.CreateNote(title, body, tags)
		if err != nil {
			return fmt.Errorf("creating note: %w", err)
		}

		if flagJSON {
			return render.JSON(os.Stdout, note)
		}

		fmt.Fprintf(os.Stderr, "Created note %s\n", note.ID)
		return nil
	},
}

func extractTitle(body string) (string, string) {
	lines := strings.SplitN(body, "\n", 2)
	first := strings.TrimSpace(lines[0])
	if strings.HasPrefix(first, "# ") {
		title := strings.TrimPrefix(first, "# ")
		rest := ""
		if len(lines) > 1 {
			rest = strings.TrimSpace(lines[1])
		}
		return title, rest
	}
	return "", body
}

func init() {
	addCmd.Flags().StringP("title", "t", "", "Note title")
	addCmd.Flags().StringP("message", "m", "", "Note body")
	addCmd.Flags().StringSlice("tag", nil, "Additional tags (key:value)")
	addCmd.Flags().Bool("no-context", false, "Skip auto-context tags")
	rootCmd.AddCommand(addCmd)
}
