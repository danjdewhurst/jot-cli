package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/context"
	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/oklog/ulid/v2"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import notes from a JSON file",
	Long:  "Import notes from a JSON export file. Only the JSON format is supported; Markdown exports are not importable.",
	Args:  cobra.ExactArgs(1),
	RunE:  runImport,
}

func init() {
	importCmd.Flags().Bool("dry-run", false, "Preview import without writing")
	importCmd.Flags().Bool("new-ids", false, "Generate new IDs instead of preserving originals")
	importCmd.Flags().Bool("no-context", false, "Skip auto-context tags")
	importCmd.Flags().StringSlice("tag", nil, "Additional tags for all imported notes (key:value)")
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	newIDs, _ := cmd.Flags().GetBool("new-ids")
	noContext, _ := cmd.Flags().GetBool("no-context")
	tagStrs, _ := cmd.Flags().GetStringSlice("tag")

	// Parse extra tags
	var extraTags []model.Tag
	for _, s := range tagStrs {
		t, err := model.ParseTag(s)
		if err != nil {
			return err
		}
		extraTags = append(extraTags, t)
	}

	// Read input
	var r io.Reader
	if args[0] == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(args[0])
		if err != nil {
			return fmt.Errorf("opening file: %w", err)
		}
		defer f.Close() //nolint:errcheck
		r = f
	}

	var envelope model.ExportEnvelope
	if err := json.NewDecoder(r).Decode(&envelope); err != nil {
		return fmt.Errorf("decoding JSON: %w", err)
	}

	if envelope.Version > model.ExportVersion {
		return fmt.Errorf("unsupported export version %d (this build supports up to %d)", envelope.Version, model.ExportVersion)
	}

	if envelope.Count != len(envelope.Notes) {
		fmt.Fprintf(os.Stderr, "Warning: envelope count (%d) does not match notes length (%d)\n", envelope.Count, len(envelope.Notes))
	}

	// Gather auto-context tags once
	var autoTags []model.Tag
	if !noContext {
		autoTags = context.AutoTags()
	}

	result := model.ImportResult{}

	for _, n := range envelope.Notes {
		if newIDs {
			n.ID = ulid.Make().String()
			now := time.Now().UTC()
			n.CreatedAt = now
			n.UpdatedAt = now
		}

		if len(autoTags) > 0 {
			n.Tags = append(n.Tags, autoTags...)
		}
		if len(extraTags) > 0 {
			n.Tags = append(n.Tags, extraTags...)
		}

		if dryRun {
			title := n.Title
			if title == "" {
				title = "(untitled)"
			}
			fmt.Fprintf(os.Stderr, "  [dry-run] %s — %s\n", n.ID, title)
			result.Created++
			continue
		}

		created, err := db.ImportNote(n)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", n.ID, err))
			continue
		}
		if created {
			result.Created++
		} else {
			result.Skipped++
		}
	}

	if flagJSON {
		return render.JSON(os.Stdout, result)
	}

	action := "Imported"
	if dryRun {
		action = "Would import"
	}
	fmt.Fprintf(os.Stderr, "%s %d notes (%d skipped)\n", action, result.Created, result.Skipped)
	if len(result.Errors) > 0 {
		fmt.Fprintf(os.Stderr, "%d errors:\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  %s\n", e)
		}
	}

	return nil
}
