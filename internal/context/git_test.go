package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectBranch(t *testing.T) {
	// Create a fake git repo
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.Mkdir(gitDir, 0o755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644)

	// Change to the temp dir
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	branch, err := DetectBranch()
	if err != nil {
		t.Fatalf("DetectBranch: %v", err)
	}
	if branch != "main" {
		t.Errorf("branch = %q, want %q", branch, "main")
	}
}

func TestDetectRepo(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.Mkdir(gitDir, 0o755)

	config := `[core]
	repositoryformatversion = 0
[remote "origin"]
	url = git@github.com:user/myproject.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	os.WriteFile(filepath.Join(gitDir, "config"), []byte(config), 0o644)

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	repo, err := DetectRepo()
	if err != nil {
		t.Fatalf("DetectRepo: %v", err)
	}
	if repo != "user/myproject" {
		t.Errorf("repo = %q, want %q", repo, "user/myproject")
	}
}

func TestDetectRepoHTTPS(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.Mkdir(gitDir, 0o755)

	config := `[remote "origin"]
	url = https://github.com/user/myproject.git
`
	os.WriteFile(filepath.Join(gitDir, "config"), []byte(config), 0o644)

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	repo, err := DetectRepo()
	if err != nil {
		t.Fatalf("DetectRepo: %v", err)
	}
	if repo != "user/myproject" {
		t.Errorf("repo = %q, want %q", repo, "user/myproject")
	}
}

func TestDetectRepoFallback(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.Mkdir(gitDir, 0o755)
	// No config file — should fall back to directory name
	os.WriteFile(filepath.Join(gitDir, "config"), []byte("[core]\n"), 0o644)

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	repo, err := DetectRepo()
	if err != nil {
		t.Fatalf("DetectRepo: %v", err)
	}
	expected := filepath.Base(dir)
	if repo != expected {
		t.Errorf("repo = %q, want %q", repo, expected)
	}
}

func TestWorktreeSupport(t *testing.T) {
	// Create a fake worktree: .git is a file pointing to another dir
	dir := t.TempDir()
	mainGitDir := filepath.Join(dir, "main-repo", ".git")
	os.MkdirAll(mainGitDir, 0o755)
	os.WriteFile(filepath.Join(mainGitDir, "HEAD"), []byte("ref: refs/heads/feature\n"), 0o644)

	worktreeDir := filepath.Join(dir, "worktree")
	os.MkdirAll(worktreeDir, 0o755)
	os.WriteFile(filepath.Join(worktreeDir, ".git"), []byte("gitdir: "+mainGitDir+"\n"), 0o644)

	orig, _ := os.Getwd()
	os.Chdir(worktreeDir)
	defer os.Chdir(orig)

	branch, err := DetectBranch()
	if err != nil {
		t.Fatalf("DetectBranch in worktree: %v", err)
	}
	if branch != "feature" {
		t.Errorf("branch = %q, want %q", branch, "feature")
	}
}

func TestNoGitRepo(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	branch, err := DetectBranch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "" {
		t.Errorf("branch = %q, want empty", branch)
	}

	repo, err := DetectRepo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo != "" {
		t.Errorf("repo = %q, want empty", repo)
	}
}

func TestRepoNameFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"git@github.com:user/repo.git", "user/repo"},
		{"https://github.com/user/repo.git", "user/repo"},
		{"https://github.com/user/repo", "user/repo"},
		{"git@gitlab.com:org/project.git", "org/project"},
	}
	for _, tt := range tests {
		got := repoNameFromURL(tt.url)
		if got != tt.want {
			t.Errorf("repoNameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}
