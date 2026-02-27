# wtw — Git Worktree workflow Helper

> **Early stages** — this is a hobby project and a work in progress. Things may
> change, break, or improve over time. If you run into a problem, please
> [open an issue](https://github.com/mehranhadidi/wtw/issues). If you have an
> idea or improvement, [PRs are very welcome](https://github.com/mehranhadidi/wtw/pulls).

## What is it?

`wtw` is a small command-line tool that makes working with Git worktrees easier.
A Git worktree lets you check out multiple branches at the same time, each in its
own directory — so you can switch between tasks without stashing or losing your place.

## Basic usage

Here's what a typical workflow looks like from start to finish:

```bash
# 1. You're on main and want to start a new feature
cd my-project

# 2. Create a worktree for your feature branch
#    This creates ../my-project-feature-login, checks out the branch,
#    and runs your .wtwrc setup script automatically
wtw feature/login

# 3. Switch to the new worktree
cd ../my-project-feature-login

# 4. Work on your feature — edit files, run the app, write tests
#    Your main branch is untouched and still running in ../my-project

# 5. Commit your changes as normal
git add .
git commit -m "Add login feature"

# 6. Push and open a pull request
git push origin feature/login

# 7. Once merged, remove the worktree
wtw done
# → confirms removal and prints: cd ../my-project

# 8. Go back to main
cd ../my-project
```

---

## Why was it created?

Modern development often means juggling multiple features at once — especially when
working with AI coding agents, where you might have several agents each working on a
different feature simultaneously. Without worktrees, those agents would be stepping on
each other's changes on the same branch. Each feature needs its own isolated branch
and working directory so changes don't conflict.

The built-in `git worktree` command solves this, but it's verbose. Creating a worktree
requires remembering the right flags, picking a sensible directory name, and then setting
up the project by hand every single time. `wtw` handles all of that in one command.

## How does it work?

Run `wtw <branch>` from inside any Git repo. It will:

1. Create a new directory next to your repo named `<repo>-<branch>`.
2. Check out the branch in that directory (creating it if it doesn't exist).
3. Automatically run your project's setup script so it's ready to work on straight away.

When you're done, run `wtw done` from inside the worktree to remove it cleanly.

## Features

| Command | Shortcut | What it does |
|---|---|---|
| `wtw <branch>` | | Create a worktree for a branch |
| `wtw <branch> <dir>` | | Create a worktree in a specific directory |
| `wtw list` | `wtw ls` | List all worktrees and their branches |
| `wtw done` | `wtw d` | Remove the current worktree |
| `wtw init` | `wtw i` | Create a sample `.wtwrc` setup script in the repo |
| `wtw run-wtwrc` | `wtw rrc` | Re-run the setup script in the current worktree |

**`-c <script>` flag** — Use a custom setup script instead of `.wtwrc`.

### Automatic project setup with `.wtwrc`

Every project has setup steps: copying `.env` files, installing dependencies,
generating keys. Without automation you'd have to remember and repeat all of this
every time you create a new worktree.

Add a `.wtwrc` file to your repo and `wtw` will run it automatically each time a
new worktree is created — so the project is fully ready to work on straight away.
Run `wtw init` to generate a commented template to get started.

The script runs inside the new worktree directory and has access to these variables:

- `$WORKTREE_PATH` — absolute path to the new worktree
- `$BRANCH_NAME` — the branch name
- `$REPO_ROOT` — absolute path to the main repo
- `$ORIGINAL_DIR` — directory where `wtw` was called from

**Laravel (PHP)**
```bash
# .wtwrc

# Copy environment file and give this worktree a unique app URL and database
cp "$REPO_ROOT/.env" .env
sed -i '' "s|APP_URL=.*|APP_URL=http://${BRANCH_NAME}.test|" .env
sed -i '' "s|DB_DATABASE=.*|DB_DATABASE=${BRANCH_NAME//-/_}|" .env

# Install dependencies and prepare the app
composer install
php artisan key:generate
php artisan migrate
```

**JavaScript (Node.js)**
```bash
# .wtwrc

# Copy environment file and give this worktree a unique port
cp "$REPO_ROOT/.env" .env
sed -i '' "s|APP_PORT=.*|APP_PORT=300${RANDOM:0:1}|" .env

# Install dependencies
npm install
```

## Installation

### Option A — Install script (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/mehranhadidi/wtw/main/install.sh | bash
```

Installs to `/usr/local/bin` by default. To use a different directory:

```bash
INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/mehranhadidi/wtw/main/install.sh | bash
```

**Upgrading** — run the same command again. The script always fetches the latest release and overwrites the existing binary.

### Option B — Build from source

```bash
git clone https://github.com/mehranhadidi/wtw
cd wtw
go build -o build/wtw .
mv build/wtw ~/.local/bin/
```

Requires [Go 1.21+](https://go.dev/dl/).

## License

MIT — see [LICENSE](LICENSE) for details.
