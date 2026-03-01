package cmd

import (
	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show note statistics",
	Long:  "Display aggregate statistics about your notes — counts, tags, and activity.",
	RunE:  runStats,
}

func runStats(cmd *cobra.Command, args []string) error {
	stats, err := db.Stats()
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()

	if flagJSON {
		return render.JSON(w, stats)
	}

	render.StatsTable(w, stats)
	return nil
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
