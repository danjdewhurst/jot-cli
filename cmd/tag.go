package cmd

import (
	"fmt"
	"os"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage tags",
}

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tags",
	RunE: func(cmd *cobra.Command, args []string) error {
		key, _ := cmd.Flags().GetString("key")

		tags, err := db.ListTags(key)
		if err != nil {
			return err
		}

		if flagJSON {
			return render.JSON(os.Stdout, tags)
		}

		render.TagTable(os.Stdout, tags)
		return nil
	},
}

var tagAddCmd = &cobra.Command{
	Use:   "add <id> <key:value>",
	Short: "Add a tag to a note",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		note, err := resolveNote(args[0])
		if err != nil {
			return err
		}

		tag, err := model.ParseTag(args[1])
		if err != nil {
			return err
		}

		if err := db.AddTag(note.ID, tag); err != nil {
			return err
		}

		if flagJSON {
			updated, _ := db.GetNote(note.ID)
			return render.JSON(os.Stdout, updated)
		}

		fmt.Fprintf(os.Stderr, "Added tag %s to %s\n", tag, note.ID)
		return nil
	},
}

var tagRmCmd = &cobra.Command{
	Use:   "rm <id> <key:value>",
	Short: "Remove a tag from a note",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		note, err := resolveNote(args[0])
		if err != nil {
			return err
		}

		tag, err := model.ParseTag(args[1])
		if err != nil {
			return err
		}

		if err := db.RemoveTag(note.ID, tag); err != nil {
			return err
		}

		if flagJSON {
			updated, _ := db.GetNote(note.ID)
			return render.JSON(os.Stdout, updated)
		}

		fmt.Fprintf(os.Stderr, "Removed tag %s from %s\n", tag, note.ID)
		return nil
	},
}

func init() {
	tagListCmd.Flags().String("key", "", "Filter tags by key")
	tagCmd.AddCommand(tagListCmd, tagAddCmd, tagRmCmd)
	rootCmd.AddCommand(tagCmd)
}
