package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/context"
	"github.com/danjdewhurst/jot-cli/internal/importer"
	"github.com/danjdewhurst/jot-cli/internal/model"
	"github.com/danjdewhurst/jot-cli/internal/render"
	"github.com/oklog/ulid/v2"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import <file|directory>",
	Short: "Import notes from a JSON file or markdown files",
	Long:  "Import notes from a JSON export file, a single markdown file, or a directory of markdown files.",
	Args:  cobra.ExactArgs(1),
	RunE:  runImport,
}

func init() {
	importCmd.Flags().Bool("dry-run", false, "Preview import without writing")
	importCmd.Flags().Bool("new-ids", false, "Generate new IDs instead of preserving originals")
	importCmd.Flags().Bool("no-context", false, "Skip auto-context tags")
	importCmd.Flags().Bool("force", false, "Skip deduplication check")
	importCmd.Flags().StringSlice("tag", nil, "Additional tags for all imported notes (key:value)")
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	noContext, _ := cmd.Flags().GetBool("no-context")
	extraTags, err := parseTags(cmd)
	if err != nil {
		return err
	}

	arg := args[0]

	// Determine import mode
	info, err := os.Stat(arg)
	if err != nil && arg != "-" {
		return fmt.Errorf("accessing %s: %w", arg, err)
	}

	if arg == "-" || (info != nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(arg), ".json")) {
		return runImportJSON(cmd, arg, dryRun, noContext, extraTags)
	}

	if info != nil && info.IsDir() {
		return runImportMarkdownDir(cmd, arg, dryRun, noContext, extraTags)
	}

	if info != nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(arg), ".md") {
		return runImportMarkdownFile(cmd, arg, dryRun, noContext, extraTags)
	}

	// Default: try JSON for backwards compatibility
	return runImportJSON(cmd, arg, dryRun, noContext, extraTags)
}

func runImportJSON(cmd *cobra.Command, arg string, dryRun, noContext bool, extraTags []model.Tag) error {
	newIDs, _ := cmd.Flags().GetBool("new-ids")

	var r io.Reader
	if arg == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(arg)
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

	return printImportResult(result, dryRun)
}

func runImportMarkdownDir(cmd *cobra.Command, dir string, dryRun, noContext bool, extraTags []model.Tag) error {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking directory: %w", err)
	}

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "No .md files found in %s\n", dir)
		return nil
	}

	fmt.Fprintf(os.Stderr, "Importing %d files...\n", len(files))

	result := model.ImportResult{}
	for _, path := range files {
		r, err := importSingleMarkdown(cmd, path, dryRun, noContext, extraTags)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		result.Created += r.Created
		result.Skipped += r.Skipped
		result.Errors = append(result.Errors, r.Errors...)
	}

	return printImportResult(result, dryRun)
}

func runImportMarkdownFile(cmd *cobra.Command, path string, dryRun, noContext bool, extraTags []model.Tag) error {
	result, err := importSingleMarkdown(cmd, path, dryRun, noContext, extraTags)
	if err != nil {
		return err
	}
	return printImportResult(result, dryRun)
}

func importSingleMarkdown(cmd *cobra.Command, path string, dryRun, noContext bool, extraTags []model.Tag) (model.ImportResult, error) {
	force, _ := cmd.Flags().GetBool("force")

	parsed, err := importer.ParseFile(path)
	if err != nil {
		return model.ImportResult{}, fmt.Errorf("parsing %s: %w", path, err)
	}

	result := model.ImportResult{}

	// Dedup check
	if !force && !dryRun {
		exists, err := db.NoteExistsByContent(parsed.Title, parsed.Body)
		if err != nil {
			return model.ImportResult{}, fmt.Errorf("checking duplicates: %w", err)
		}
		if exists {
			result.Skipped++
			return result, nil
		}
	}

	// Build note
	now := time.Now().UTC()
	id := ulid.Make().String()

	createdAt := now
	if parsed.CreatedAt != nil {
		createdAt = *parsed.CreatedAt
	} else {
		// Use file modification time
		if info, err := os.Stat(path); err == nil {
			createdAt = info.ModTime().UTC()
		}
	}

	var tags []model.Tag
	tags = append(tags, parsed.Tags...)

	if !noContext {
		tags = append(tags, context.AutoTags()...)
	}
	tags = append(tags, extraTags...)

	n := model.Note{
		ID:        id,
		Title:     parsed.Title,
		Body:      parsed.Body,
		CreatedAt: createdAt,
		UpdatedAt: now,
		Tags:      tags,
	}

	if dryRun {
		title := n.Title
		if title == "" {
			title = "(untitled)"
		}
		fmt.Fprintf(os.Stderr, "  [dry-run] %s — %s\n", filepath.Base(path), title)
		result.Created++
		return result, nil
	}

	created, err := db.ImportNote(n)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
		return result, nil
	}
	if created {
		result.Created++
	} else {
		result.Skipped++
	}

	return result, nil
}

func printImportResult(result model.ImportResult, dryRun bool) error {
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
