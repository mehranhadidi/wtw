#!/usr/bin/env bash
#
# Tags a new release and pushes it to GitHub.
# Usage: ./release.sh <version>
# Example: ./release.sh 1.0.0

set -euo pipefail

VERSION="${1:-}"

if [[ -z "$VERSION" ]]; then
  echo "Usage: ./release.sh <version>  (e.g. ./release.sh 1.0.0)"
  exit 1
fi

TAG="v$VERSION"

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

# Pull latest changes
git pull --ff-only origin main

echo "Tagging $TAG and pushing..."
git tag "$TAG"
git push origin main
git push origin "$TAG"

echo "Done. GitHub Actions will build and publish the release automatically."
echo "Track progress at: https://github.com/mehranhadidi/wtw/actions"
