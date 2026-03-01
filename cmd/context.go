package cmd

import (
	"fmt"

	appctx "github.com/danjdewhurst/jot-cli/internal/context"
	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Show detected environment context",
	Long:  "Print the auto-detected context tags (folder, git repo, branch) without creating a note.",
	RunE:  runContext,
}

func runContext(cmd *cobra.Command, args []string) error {
	tags := appctx.AutoTags()
	w := cmd.OutOrStdout()

	if flagJSON {
		return render.JSON(w, tags)
	}

	// Always show all three keys, with "(none)" for missing values
	keys := []string{"folder", "git_repo", "git_branch"}
	tagMap := make(map[string]string, len(tags))
	for _, t := range tags {
		tagMap[t.Key] = t.Value
	}

	var allTags []model.Tag
	for _, k := range keys {
		v := tagMap[k]
		if v == "" {
			v = "(none)"
		}
		allTags = append(allTags, model.Tag{Key: k, Value: v})
	}

	for _, t := range allTags {
		_, _ = fmt.Fprintf(w, "%-12s %s\n", t.Key+":", t.Value)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(contextCmd)
}
