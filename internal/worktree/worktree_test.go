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

func TestRunScript_EnvVars(t *testing.T) {
	dir := t.TempDir()
	worktreePath := filepath.Join(dir, "myapp-feature-x")
	repoRoot := filepath.Join(dir, "myapp")
	if err := os.MkdirAll(worktreePath, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a script that prints the env vars we care about.
	script := filepath.Join(dir, "check.sh")
	if err := os.WriteFile(script, []byte("echo \"$WORKTREE_NAME|$REPO_NAME\""), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := captureRunScript(script, worktreePath, "feature-x", repoRoot, dir)
	if err != nil {
		t.Fatalf("RunScript: %v", err)
	}
	got := strings.TrimSpace(out)
	want := "myapp-feature-x|myapp"
	if got != want {
		t.Errorf("env vars = %q, want %q", got, want)
	}
}

// captureRunScript runs RunScript but captures stdout instead of inheriting it.
func captureRunScript(scriptPath, worktreePath, branchName, repoRoot, originalDir string) (string, error) {
	cmd := execCommand("bash", scriptPath)
	cmd.Dir = worktreePath
	var buf strings.Builder
	cmd.Stdout = &buf
	cmd.Env = append(os.Environ(),
		"WORKTREE_PATH="+worktreePath,
		"WORKTREE_NAME="+filepath.Base(worktreePath),
		"BRANCH_NAME="+branchName,
		"REPO_NAME="+filepath.Base(repoRoot),
		"REPO_ROOT="+repoRoot,
		"ORIGINAL_DIR="+originalDir,
	)
	err := cmd.Run()
	return buf.String(), err
}

func execCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

func writeEnvFile(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readEnvFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestEnvSet_ReplacesExisting(t *testing.T) {
	dir := t.TempDir()
	path := writeEnvFile(t, dir, "DEBUG=true\nAPP_URL=http://localhost\n")

	if err := EnvSet(EnvSetConfig{File: path, Pairs: []string{"DEBUG=false"}}); err != nil {
		t.Fatal(err)
	}

	got := readEnvFile(t, path)
	want := "DEBUG=false\nAPP_URL=http://localhost\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEnvSet_AppendsNew(t *testing.T) {
	dir := t.TempDir()
	path := writeEnvFile(t, dir, "DEBUG=true\n")

	if err := EnvSet(EnvSetConfig{File: path, Pairs: []string{"NEWKEY=hello"}}); err != nil {
		t.Fatal(err)
	}

	got := readEnvFile(t, path)
	want := "DEBUG=true\nNEWKEY=hello\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEnvSet_MultipleValues(t *testing.T) {
	dir := t.TempDir()
	path := writeEnvFile(t, dir, "DEBUG=true\nAPP_URL=http://localhost\n")

	err := EnvSet(EnvSetConfig{
		File:  path,
		Pairs: []string{"DEBUG=false", "APP_URL=http://myapp-feat.test", "DB_NAME=mydb"},
	})
	if err != nil {
		t.Fatal(err)
	}

	got := readEnvFile(t, path)
	want := "DEBUG=false\nAPP_URL=http://myapp-feat.test\nDB_NAME=mydb\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEnvSet_PreservesComments(t *testing.T) {
	dir := t.TempDir()
	path := writeEnvFile(t, dir, "# app config\nDEBUG=true\n\n# url\nAPP_URL=http://localhost\n")

	if err := EnvSet(EnvSetConfig{File: path, Pairs: []string{"DEBUG=false"}}); err != nil {
		t.Fatal(err)
	}

	got := readEnvFile(t, path)
	want := "# app config\nDEBUG=false\n\n# url\nAPP_URL=http://localhost\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEnvSet_ValueContainsEquals(t *testing.T) {
	dir := t.TempDir()
	path := writeEnvFile(t, dir, "TOKEN=old\n")

	if err := EnvSet(EnvSetConfig{File: path, Pairs: []string{"TOKEN=a=b=c"}}); err != nil {
		t.Fatal(err)
	}

	got := readEnvFile(t, path)
	want := "TOKEN=a=b=c\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEnvSet_MissingFile(t *testing.T) {
	err := EnvSet(EnvSetConfig{File: "/nonexistent/.env", Pairs: []string{"K=V"}})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestEnvSet_InvalidPair(t *testing.T) {
	dir := t.TempDir()
	path := writeEnvFile(t, dir, "K=V\n")

	err := EnvSet(EnvSetConfig{File: path, Pairs: []string{"NOEQUALS"}})
	if err == nil {
		t.Fatal("expected error for pair without '='")
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
