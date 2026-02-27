package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"wtw/internal/ui"
)

// setupRepo creates a fresh git repo under ./tmp/, chdirs into it, and
// registers cleanup. Using a local directory makes failed test artifacts easy
// to inspect; tmp/ is gitignored.
func setupRepo(t *testing.T) string {
	t.Helper()

	if err := os.MkdirAll("tmp", 0o755); err != nil {
		t.Fatal(err)
	}
	rel, err := os.MkdirTemp("tmp", "wtw-test-*")
	if err != nil {
		t.Fatal(err)
	}
	// Resolve to absolute so paths stay valid after os.Chdir.
	dir, err := filepath.Abs(rel)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	// Clean git environment so system/global config doesn't interfere.
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("HOME", dir)
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	run("commit", "--allow-empty", "-m", "initial")

	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	return dir
}

func TestSanitizeBranch(t *testing.T) {
	cases := []struct{ in, want string }{
		{"feature/my-thing", "feature-my-thing"},
		{"fix/bug--123", "fix-bug-123"},
		{"-leading", "leading"},
		{"trailing-", "trailing"},
		{"a b c", "a-b-c"},
		{"normal", "normal"},
	}
	for _, tc := range cases {
		got := SanitizeBranch(tc.in)
		if got != tc.want {
			t.Errorf("SanitizeBranch(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCreate_NewBranch(t *testing.T) {
	repoRoot := setupRepo(t)
	repoName := filepath.Base(repoRoot)

	cfg := CreateConfig{
		BranchName:  "test-branch",
		RepoRoot:    repoRoot,
		RepoName:    repoName,
		OriginalDir: repoRoot,
	}

	if err := Create(cfg); err != nil {
		t.Fatalf("Create: %v", err)
	}

	want := filepath.Join(filepath.Dir(repoRoot), repoName+"-test-branch")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("worktree dir not found at %s: %v", want, err)
	}
}

func TestCreate_ExistingBranch(t *testing.T) {
	repoRoot := setupRepo(t)
	repoName := filepath.Base(repoRoot)

	// Create branch first
	cmd := exec.Command("git", "branch", "existing")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git branch: %v\n%s", err, out)
	}

	cfg := CreateConfig{
		BranchName:  "existing",
		RepoRoot:    repoRoot,
		RepoName:    repoName,
		OriginalDir: repoRoot,
	}

	if err := Create(cfg); err != nil {
		t.Fatalf("Create: %v", err)
	}

	want := filepath.Join(filepath.Dir(repoRoot), repoName+"-existing")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("worktree dir not found at %s: %v", want, err)
	}
}

func TestCreate_SanitizesBranchName(t *testing.T) {
	repoRoot := setupRepo(t)
	repoName := filepath.Base(repoRoot)

	cfg := CreateConfig{
		BranchName:  "feature/x",
		RepoRoot:    repoRoot,
		RepoName:    repoName,
		OriginalDir: repoRoot,
	}

	if err := Create(cfg); err != nil {
		t.Fatalf("Create: %v", err)
	}

	want := filepath.Join(filepath.Dir(repoRoot), repoName+"-feature-x")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("worktree dir not found at %s: %v", want, err)
	}
}

func TestCreate_EmptyBranchName(t *testing.T) {
	repoRoot := setupRepo(t)
	repoName := filepath.Base(repoRoot)

	// Inject empty input so prompt returns ""
	ui.SetReader(strings.NewReader("\n"))

	cfg := CreateConfig{
		BranchName:  "",
		RepoRoot:    repoRoot,
		RepoName:    repoName,
		OriginalDir: repoRoot,
	}

	err := Create(cfg)
	if err == nil {
		t.Fatal("expected error for empty branch name")
	}
}
