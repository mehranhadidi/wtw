# wtw — Claude Project Context

This file gives AI agents working on this project the context they need to
contribute effectively.

## What the project is

`wtw` is a CLI tool that simplifies Git worktree management. It wraps
`git worktree` with a friendly interface, automatic directory naming, and
optional project setup scripts (`.wtwrc`).

## Architecture

```
main.go                      Entry point — calls cmd.Execute()
cmd/                         Thin cobra commands, no business logic
  root.go                    Root command + Execute(), -c flag
  create.go                  Default action: create a worktree
  list.go                    List all worktrees
  done.go                    Remove current worktree
  init.go                    Create sample .wtwrc
  run.go                     Re-run setup script
internal/git/                Git subprocess wrappers and pure parsers
internal/ui/                 Terminal output and user input
internal/worktree/           All business logic
```

**Dependency flow:** `cmd` → `internal/worktree` → `internal/git`
**No circular dependencies. `internal/` is not importable outside the module.**

## Build & run

```bash
go build -o build/wtw .    # build binary
build/wtw --help           # run
```

## Testing

```bash
go test ./...
```

Integration tests in `internal/worktree/worktree_test.go` create real git repos
under `internal/worktree/tmp/` (gitignored). Tests must not call `t.Parallel()`
because they use `os.Chdir`.

## Conventions

### Commit messages — Conventional Commits

```
feat: add wtw list command
fix: pass mainRepoRoot to RunScript in RunSetup
docs: update README with list command
test: add ParseWorktrees unit test
refactor: remove fileExists helper
chore: update module path to wtw
```

### Go style

- Keep `cmd/` files thin — resolve flags/args, build a config struct, delegate to `internal/worktree`
- Exported functions get doc comments; unexported helpers do not unless non-obvious
- Errors are wrapped with `fmt.Errorf("context: %w", err)` when adding context, or returned as-is when not
- No `fileExists`-style single-use helpers — inline `os.Stat` directly

### Adding a new subcommand

1. Create `cmd/<name>.go` with a `cobra.Command` and `init()` that calls `rootCmd.AddCommand`
2. Add the business logic function to `internal/worktree/worktree.go`
3. Add a test in `internal/worktree/worktree_test.go`
4. Add the command to the features table in `README.md`

## Git workflow

- **`main`** is the only long-lived branch — all work branches off it
- Use a worktree per feature: `wtw feature/my-thing` from within this repo
- Open a PR into `main` when ready
- Delete the worktree with `wtw done` after merging

## Releasing

```bash
./release.sh 1.2.0
```

This tags `v1.2.0`, pushes to GitHub, and triggers goreleaser via GitHub Actions.
Binaries are published to GitHub Releases automatically.
