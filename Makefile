LINT_VERSION ?= v2.10.1
LOCAL_BIN_DIR ?= $(HOME)/.local/bin

.PHONY: test lint

test:
	go test ./...

lint:
	LINT_VERSION=$(LINT_VERSION) LOCAL_BIN_DIR=$(LOCAL_BIN_DIR) ./scripts/lint.sh
