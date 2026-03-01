package editor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func editorCmd() string {
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	return "vi"
}

// Edit opens the user's editor with initial content and returns the edited text.
func Edit(initial string) (string, error) {
	f, err := os.CreateTemp("", "jot-*.md")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(f.Name())

	if _, err := f.WriteString(initial); err != nil {
		f.Close()
		return "", fmt.Errorf("writing temp file: %w", err)
	}
	f.Close()

	editor := editorCmd()
	parts := strings.Fields(editor)
	parts = append(parts, f.Name())

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	data, err := os.ReadFile(f.Name())
	if err != nil {
		return "", fmt.Errorf("reading edited file: %w", err)
	}
	return string(data), nil
}
