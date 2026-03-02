#!/usr/bin/env bash

set -euo pipefail

LINT_VERSION="${LINT_VERSION:-v2.10.1}"
LOCAL_BIN_DIR="${LOCAL_BIN_DIR:-$HOME/.local/bin}"
LINT_BIN="$LOCAL_BIN_DIR/golangci-lint"

mkdir -p "$LOCAL_BIN_DIR"

if [[ ! -x "$LINT_BIN" ]]; then
  echo "Installing golangci-lint $LINT_VERSION to $LINT_BIN"
  GOBIN="$LOCAL_BIN_DIR" go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${LINT_VERSION}"
fi

echo "Running golangci-lint ($LINT_VERSION)"
"$LINT_BIN" run ./...
