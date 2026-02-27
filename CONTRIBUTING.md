# Contributing

Thanks for taking the time to contribute! This is a hobby project, so there are
no strict rules — just a few guidelines to keep things consistent.

## Getting started

```bash
git clone https://github.com/mehranhadidi/wtw
cd wtw
go build -o build/wtw .
```

Requires Go 1.21+.

## Making changes

1. Fork the repo and create a branch: `git checkout -b feat/my-thing`
   Or use the tool itself: `wtw feat/my-thing`
2. Make your changes
3. Add or update tests if relevant
4. Run `go test ./...` and make sure everything passes
5. Open a pull request

## Commit messages

This project uses [Conventional Commits](https://www.conventionalcommits.org):

| Prefix | When to use |
|---|---|
| `feat:` | A new feature |
| `fix:` | A bug fix |
| `docs:` | Documentation only |
| `test:` | Adding or updating tests |
| `refactor:` | Code change that isn't a fix or feature |
| `chore:` | Build, config, tooling changes |

Example: `feat: add wtw list command`

## Reporting issues

Open an issue at https://github.com/mehranhadidi/wtw/issues with as much
detail as you can — what you ran, what you expected, and what actually happened.
