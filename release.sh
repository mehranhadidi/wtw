#!/usr/bin/env bash
#
# Tags a new release and pushes it to GitHub.
#
# Usage:
#   ./release.sh          # bumps patch:  0.0.2 → 0.0.3
#   ./release.sh minor    # bumps minor:  0.0.2 → 0.1.0
#   ./release.sh major    # bumps major:  0.0.2 → 1.0.0

set -euo pipefail

BUMP="${1:-patch}"

# Make sure the working tree is clean
if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "You have uncommitted changes. Please commit or stash them first."
  exit 1
fi

# Make sure we're on main
BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$BRANCH" != "main" ]]; then
  echo "You're on '$BRANCH'. Releases should be tagged from main."
  exit 1
fi

# Get the latest version tag (default to 0.0.0 if none exists)
LATEST=$(git tag --list "v*" --sort=-version:refname | head -1)
LATEST="${LATEST:-v0.0.0}"
LATEST="${LATEST#v}"  # strip leading "v"

# Parse major.minor.patch
IFS='.' read -r MAJOR MINOR PATCH <<< "$LATEST"

case "$BUMP" in
  major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
  minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
  patch) PATCH=$((PATCH + 1)) ;;
  *)
    echo "Unknown bump type '$BUMP'. Use: patch, minor, or major."
    exit 1
    ;;
esac

TAG="v${MAJOR}.${MINOR}.${PATCH}"

echo "Current version : v$LATEST"
echo "New version     : $TAG"
echo ""
read -rp "Confirm release $TAG? [y/N] " CONFIRM
[[ "$CONFIRM" =~ ^[Yy]$ ]] || { echo "Aborted."; exit 0; }

# Pull latest changes
git pull --ff-only origin main

git tag "$TAG"
git push origin main
git push origin "$TAG"

echo ""
echo "Done. GitHub Actions will build and publish the release automatically."
echo "Track progress at: https://github.com/mehranhadidi/wtw/actions"
