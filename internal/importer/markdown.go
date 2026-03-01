package importer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/danjdewhurst/jot-cli/internal/model"
	"gopkg.in/yaml.v3"
)

// ParsedNote holds the result of parsing a markdown file.
type ParsedNote struct {
	Title     string
	Body      string
	Tags      []model.Tag
	CreatedAt *time.Time
}

// frontmatter represents the YAML frontmatter structure.
type frontmatter struct {
	Title   string   `yaml:"title"`
	Tags    []string `yaml:"tags"`
	Created string   `yaml:"created"`
}

// ParseFile parses a single markdown file at the given path.
func ParseFile(path string) (ParsedNote, error) {
	f, err := os.Open(path)
	if err != nil {
		return ParsedNote{}, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close() //nolint:errcheck

	return ParseReader(f)
}

// ParseReader parses markdown content from a reader.
func ParseReader(r io.Reader) (ParsedNote, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return ParsedNote{}, fmt.Errorf("reading input: %w", err)
	}

	content := string(data)
	var result ParsedNote

	if strings.HasPrefix(content, "---\n") {
		result = parseFrontmatter(content)
	} else {
		result = parseNoFrontmatter(content)
	}

	return result, nil
}

func parseFrontmatter(content string) ParsedNote {
	// Find closing delimiter
	rest := content[4:] // skip opening "---\n"
	idx := strings.Index(rest, "\n---\n")
	if idx < 0 {
		// No closing delimiter — treat as plain markdown
		return parseNoFrontmatter(content)
	}

	yamlBlock := rest[:idx]
	body := rest[idx+5:] // skip "\n---\n"

	var fm frontmatter
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		// Malformed YAML — fall back to treating whole content as body
		return parseNoFrontmatter(content)
	}

	var result ParsedNote
	result.Title = fm.Title
	result.Body = strings.TrimLeft(body, "\n")

	// Parse tags
	for _, s := range fm.Tags {
		tag, err := model.ParseTag(s)
		if err != nil {
			continue // skip invalid tags
		}
		result.Tags = append(result.Tags, tag)
	}

	// Parse created timestamp
	if fm.Created != "" {
		if t, err := time.Parse(time.RFC3339, fm.Created); err == nil {
			result.CreatedAt = &t
		} else if t, err := time.Parse("2006-01-02", fm.Created); err == nil {
			utc := t.UTC()
			result.CreatedAt = &utc
		}
	}

	return result
}

func parseNoFrontmatter(content string) ParsedNote {
	var result ParsedNote
	scanner := bufio.NewScanner(strings.NewReader(content))

	// Look for first # heading
	var lines []string
	titleFound := false
	for scanner.Scan() {
		line := scanner.Text()
		if !titleFound && strings.HasPrefix(line, "# ") {
			result.Title = strings.TrimPrefix(line, "# ")
			titleFound = true
			continue
		}
		lines = append(lines, line)
	}

	body := strings.Join(lines, "\n")
	// Trim leading blank lines (from after the title)
	body = strings.TrimLeft(body, "\n")
	// Preserve trailing newline if present in original
	if len(body) > 0 && !strings.HasSuffix(body, "\n") && strings.HasSuffix(content, "\n") {
		body += "\n"
	}
	result.Body = body

	return result
}
