package context

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// findGitDir walks up from dir to find .git (file or directory).
// Returns the resolved .git directory path, or empty string if not found.
func findGitDir(dir string) string {
	for {
		gitPath := filepath.Join(dir, ".git")
		info, err := os.Lstat(gitPath)
		if err == nil {
			if info.IsDir() {
				return gitPath
			}
			// Worktree: .git is a file containing "gitdir: <path>"
			data, err := os.ReadFile(gitPath)
			if err == nil {
				line := strings.TrimSpace(string(data))
				if after, ok := strings.CutPrefix(line, "gitdir: "); ok {
					if filepath.IsAbs(after) {
						return after
					}
					return filepath.Join(dir, after)
				}
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func DetectBranch() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	gitDir := findGitDir(wd)
	if gitDir == "" {
		return "", nil
	}

	head, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return "", nil
	}

	line := strings.TrimSpace(string(head))
	if after, ok := strings.CutPrefix(line, "ref: refs/heads/"); ok {
		return after, nil
	}

	// Detached HEAD — return short hash
	if len(line) >= 8 {
		return line[:8], nil
	}
	return line, nil
}

func DetectRepo() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	gitDir := findGitDir(wd)
	if gitDir == "" {
		return "", nil
	}

	// Try to parse origin remote URL from config
	configPath := filepath.Join(gitDir, "config")
	repo := parseOriginURL(configPath)
	if repo != "" {
		return repo, nil
	}

	// Fallback: directory name containing .git
	return filepath.Base(filepath.Dir(gitDir)), nil
}

func parseOriginURL(configPath string) string {
	f, err := os.Open(configPath)
	if err != nil {
		return ""
	}
	defer f.Close() //nolint:errcheck // best-effort close on read-only file

	scanner := bufio.NewScanner(f)
	inOrigin := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == `[remote "origin"]` {
			inOrigin = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inOrigin = false
			continue
		}
		if inOrigin {
			key, val, ok := strings.Cut(line, "=")
			if ok && strings.TrimSpace(key) == "url" {
				return repoNameFromURL(strings.TrimSpace(val))
			}
		}
	}
	return ""
}

func repoNameFromURL(rawURL string) string {
	// Handle SSH: git@github.com:user/repo.git
	if _, after, ok := strings.Cut(rawURL, ":"); ok && !strings.HasPrefix(rawURL, "http") {
		return strings.TrimSuffix(after, ".git")
	}

	// Handle HTTPS: https://github.com/user/repo.git
	rawURL = strings.TrimSuffix(rawURL, ".git")
	parts := strings.Split(rawURL, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}
	return rawURL
}
