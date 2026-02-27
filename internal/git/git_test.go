package git

import "testing"

func TestParseWorktreeForBranch(t *testing.T) {
	porcelain := `worktree /home/user/project
HEAD abc123
branch refs/heads/main

worktree /home/user/project-feature-x
HEAD def456
branch refs/heads/feature/x

worktree /home/user/project-detached
HEAD ghi789
detached

`
	tests := []struct {
		input  string
		branch string
		want   string
	}{
		{porcelain, "main", "/home/user/project"},
		{porcelain, "feature/x", "/home/user/project-feature-x"},
		{porcelain, "nonexistent", ""},
		{"", "main", ""},
	}

	for _, tc := range tests {
		got := ParseWorktreeForBranch(tc.input, tc.branch)
		if got != tc.want {
			t.Errorf("ParseWorktreeForBranch(%q) = %q, want %q", tc.branch, got, tc.want)
		}
	}
}
