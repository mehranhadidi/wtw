// Package git wraps git subprocesses and provides pure parsers for their output.
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Output runs a git command and returns trimmed stdout.
func Output(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	return strings.TrimSpace(string(out)), err
}

// Run runs a git command in dir, streaming stdout/stderr to the terminal.
func Run(dir string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RepoRoot returns the absolute path of the repository root.
func RepoRoot() (string, error) {
	return Output("rev-parse", "--show-toplevel")
}

// RequireWorktree asserts the cwd is inside a linked worktree, not the main repo.
// Returns (worktreeRoot, mainRepoRoot, error).
//
// How it detects worktrees: --git-common-dir always resolves to the main .git
// directory, while --absolute-git-dir resolves to .git/worktrees/<name> for a
// linked worktree. If they're the same path, we're in the main repo.
func RequireWorktree(subcommand string) (string, string, error) {
	mainRepo, err := Output("rev-parse", "--git-common-dir")
	if err != nil {
		return "", "", err
	}
	mainRepo, _ = filepath.Abs(mainRepo)

	gitDir, err := Output("rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", "", err
	}
	gitDir, _ = filepath.Abs(gitDir)

	if mainRepo == gitDir {
		return "", "", fmt.Errorf("'wtw %s' must be run inside a worktree, not the main repo", subcommand)
	}

	worktreeRoot, err := RepoRoot()
	if err != nil {
		return "", "", err
	}

	return worktreeRoot, filepath.Dir(mainRepo), nil
}

// Worktree holds the path and branch of a single git worktree entry.
// Branch is empty if the worktree is in a detached HEAD state.
type Worktree struct {
	Path   string
	Branch string
}

// ListWorktrees returns all worktrees for the repo at repoRoot.
func ListWorktrees(repoRoot string) ([]Worktree, error) {
	out, err := exec.Command("git", "-C", repoRoot, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil, err
	}
	return ParseWorktrees(string(out)), nil
}

// ParseWorktrees parses `git worktree list --porcelain` output into a slice of Worktree.
// Exported so tests can call it directly without running git.
func ParseWorktrees(porcelain string) []Worktree {
	var worktrees []Worktree
	var current Worktree
	for _, line := range strings.Split(porcelain, "\n") {
		if path, ok := strings.CutPrefix(line, "worktree "); ok {
			current = Worktree{Path: strings.TrimSpace(path)}
		} else if branch, ok := strings.CutPrefix(line, "branch refs/heads/"); ok {
			current.Branch = strings.TrimSpace(branch)
		} else if line == "" && current.Path != "" {
			worktrees = append(worktrees, current)
			current = Worktree{}
		}
	}
	return worktrees
}

// WorktreeForBranch returns the worktree path checked out on branch, or "".
func WorktreeForBranch(repoRoot, branch string) string {
	out, err := exec.Command("git", "-C", repoRoot, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return ""
	}
	return ParseWorktreeForBranch(string(out), branch)
}

// ParseWorktreeForBranch parses `git worktree list --porcelain` output and
// returns the path of the worktree checked out on branch, or "".
// Exported so tests can call it directly without running git.
//
// Each stanza looks like:
//
//	worktree /path/to/tree
//	HEAD abc123
//	branch refs/heads/<name>
func ParseWorktreeForBranch(porcelain, branch string) string {
	var currentPath string
	for _, line := range strings.Split(porcelain, "\n") {
		if path, ok := strings.CutPrefix(line, "worktree "); ok {
			currentPath = strings.TrimSpace(path)
		} else if strings.TrimSpace(line) == "branch refs/heads/"+branch {
			return currentPath
		}
	}
	return ""
}

// IsRegisteredWorktree returns true if path is registered as a worktree in root.
func IsRegisteredWorktree(repoRoot, path string) bool {
	out, err := exec.Command("git", "-C", repoRoot, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "worktree "+path+"\n")
}

// BranchExists returns true if the branch exists locally.
func BranchExists(repoRoot, branch string) bool {
	out, err := exec.Command("git", "-C", repoRoot, "branch", "--list", branch).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

// AddWorktree creates a new worktree at path on branch (creating branch if needed).
func AddWorktree(repoRoot, path, branch string) error {
	if BranchExists(repoRoot, branch) {
		return Run(repoRoot, "worktree", "add", path, branch)
	}
	return Run(repoRoot, "worktree", "add", "-b", branch, path)
}

// RemoveWorktree removes a worktree (force).
func RemoveWorktree(mainRepoRoot, path string) error {
	return Run(mainRepoRoot, "worktree", "remove", "--force", path)
}
