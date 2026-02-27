// Package worktree contains the business logic for wtw.
// The cmd layer only parses flags/args and delegates here, keeping this
// package testable without going through cobra.
package worktree

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"wtw/internal/git"
	"wtw/internal/ui"
)

// wtwrcTemplate is written by Init.
const wtwrcTemplate = `# .wtwrc — wtw setup script
#
# Runs inside the new worktree directory when you create one with ` + "`wtw`" + `.
# This file is intentionally not executable — it is run by wtw only.
#
# Available variables:
#   $WORKTREE_PATH   absolute path to the new worktree
#   $BRANCH_NAME     the branch name
#   $REPO_ROOT       absolute path to the main repo
#   $ORIGINAL_DIR    directory where ` + "`wtw`" + ` was called from

# Copy environment files from the main repo
# cp "$REPO_ROOT/.env" .env
# cp "$REPO_ROOT/.env.local" .env.local

# Use $BRANCH_NAME to generate unique values per worktree, for example:
# echo "APP_URL=http://${BRANCH_NAME}.test" >> .env
# echo "DB_DATABASE=${BRANCH_NAME//-/_}" >> .env

# Install dependencies
# npm install
# composer install
# bundle install

# Generate keys or secrets
# php artisan key:generate

echo "Worktree ready: $BRANCH_NAME"
`

var reInvalid = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

// SanitizeBranch converts a branch name into a safe directory-name component.
// "/" → "-", invalid chars → "-", consecutive dashes collapsed, leading/trailing dashes stripped.
func SanitizeBranch(branch string) string {
	s := strings.ReplaceAll(branch, "/", "-")
	s = reInvalid.ReplaceAllString(s, "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

// CreateConfig holds all inputs for Create.
type CreateConfig struct {
	BranchName  string // may be empty (will prompt)
	BaseDir     string // may be empty (uses parent of repo root)
	SetupScript string // may be empty
	RepoRoot    string
	RepoName    string
	OriginalDir string
}

// Create creates a new worktree for the given branch.
func Create(cfg CreateConfig) error {
	branchName := cfg.BranchName
	if branchName == "" {
		branchName = ui.Ask("Branch name:")
	}
	if branchName == "" {
		return errors.New("branch name cannot be empty")
	}
	if strings.ContainsAny(branchName, " \t\n") {
		return errors.New("branch name cannot contain spaces")
	}

	worktreeDirname := cfg.RepoName + "-" + SanitizeBranch(branchName)

	var worktreePath string
	if cfg.BaseDir != "" {
		if err := os.MkdirAll(cfg.BaseDir, 0o755); err != nil {
			return fmt.Errorf("failed to create base directory: %w", err)
		}
		abs, err := filepath.Abs(cfg.BaseDir)
		if err != nil {
			return err
		}
		worktreePath = filepath.Join(abs, worktreeDirname)
	} else {
		worktreePath = filepath.Join(filepath.Dir(cfg.RepoRoot), worktreeDirname)
	}

	// Check for existing worktree at path
	if _, err := os.Stat(worktreePath); err == nil {
		if git.IsRegisteredWorktree(cfg.RepoRoot, worktreePath) {
			return fmt.Errorf("worktree already exists at: %s", worktreePath)
		}
		if !ui.Confirm("Directory already exists. Remove and recreate? [y/N]", "N") {
			return errors.New("aborted")
		}
		if err := os.RemoveAll(worktreePath); err != nil {
			return fmt.Errorf("failed to remove directory: %w", err)
		}
	}

	// Check if branch is already checked out elsewhere
	if existing := git.WorktreeForBranch(cfg.RepoRoot, branchName); existing != "" {
		return fmt.Errorf("branch %q already checked out at: %s", branchName, existing)
	}

	if err := git.AddWorktree(cfg.RepoRoot, worktreePath, branchName); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	if cfg.SetupScript != "" {
		if ui.Confirm("Found "+filepath.Base(cfg.SetupScript)+" — run it? [Y/n]", "Y") {
			if err := RunScript(cfg.SetupScript, worktreePath, branchName, cfg.RepoRoot, cfg.OriginalDir); err != nil {
				ui.Error("setup script failed. Retry: cd " + worktreePath + " && bash " + cfg.SetupScript)
			}
		}
	}

	ui.Success("Worktree ready.")
	ui.PrintCmd("cd " + worktreePath)
	return nil
}

// RemoveConfig holds inputs for Remove.
type RemoveConfig struct {
	WorktreeRoot string
	MainRepoRoot string
}

// Remove removes the current worktree after user confirmation.
func Remove(cfg RemoveConfig) error {
	if !ui.Confirm("Remove this worktree ("+cfg.WorktreeRoot+")? [y/N]", "N") {
		return errors.New("aborted")
	}
	if err := git.RemoveWorktree(cfg.MainRepoRoot, cfg.WorktreeRoot); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}
	ui.Success("Worktree removed.")
	ui.PrintCmd("cd " + cfg.MainRepoRoot)
	return nil
}

// RunSetupConfig holds inputs for RunSetup.
type RunSetupConfig struct {
	SetupScript  string // may be empty (falls back to .wtwrc)
	WorktreeRoot string
	MainRepoRoot string
	OriginalDir  string
}

// RunSetup re-runs the setup script in the current worktree.
func RunSetup(cfg RunSetupConfig) error {
	scriptPath := cfg.SetupScript
	if scriptPath == "" {
		candidate := filepath.Join(cfg.MainRepoRoot, ".wtwrc")
		if _, err := os.Stat(candidate); err == nil {
			scriptPath = candidate
		}
	} else {
		if _, err := os.Stat(scriptPath); err != nil {
			return fmt.Errorf("setup script not found: %s", scriptPath)
		}
	}
	if scriptPath == "" {
		return errors.New("no .wtwrc found")
	}

	branchName, _ := git.Output("branch", "--show-current")
	if err := RunScript(scriptPath, cfg.WorktreeRoot, branchName, cfg.MainRepoRoot, cfg.OriginalDir); err != nil {
		return fmt.Errorf("setup script failed: %w", err)
	}
	ui.Success("Done.")
	return nil
}

// List prints all worktrees for the current repo, marking the current one.
func List(repoRoot string) error {
	worktrees, err := git.ListWorktrees(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	cwd, _ := os.Getwd()

	for _, wt := range worktrees {
		branch := wt.Branch
		if branch == "" {
			branch = "(detached)"
		}
		if wt.Path == cwd {
			ui.PrintCmd(wt.Path + "  " + branch + "  ← current")
		} else {
			fmt.Printf("%s  %s\n", wt.Path, branch)
		}
	}
	return nil
}

// Init creates a sample .wtwrc in repoRoot.
func Init(repoRoot string) error {
	rcPath := filepath.Join(repoRoot, ".wtwrc")
	if _, err := os.Stat(rcPath); err == nil {
		return errors.New(".wtwrc already exists in this repo")
	}
	if err := os.WriteFile(rcPath, []byte(wtwrcTemplate), 0o644); err != nil {
		return fmt.Errorf("failed to create .wtwrc: %w", err)
	}
	ui.Success(".wtwrc created — edit it to define your setup steps.")
	return nil
}

// RunScript executes a bash script in worktreePath with the standard env vars.
func RunScript(scriptPath, worktreePath, branchName, repoRoot, originalDir string) error {
	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = worktreePath
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"WORKTREE_PATH="+worktreePath,
		"BRANCH_NAME="+branchName,
		"REPO_ROOT="+repoRoot,
		"ORIGINAL_DIR="+originalDir,
	)
	return cmd.Run()
}

// ResolveSetupScript resolves the effective setup script path.
// customSetup is the -c flag value (may be ""). repoRoot is the main repo root.
func ResolveSetupScript(customSetup, repoRoot string) (string, error) {
	if customSetup != "" {
		if _, err := os.Stat(customSetup); err != nil {
			return "", fmt.Errorf("setup script not found: %s", customSetup)
		}
		return customSetup, nil
	}
	rc := filepath.Join(repoRoot, ".wtwrc")
	if _, err := os.Stat(rc); err == nil {
		return rc, nil
	}
	return "", nil
}
