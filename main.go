package main

import (
	"fmt"
	"os"

	"wtw/cmd"
)

// version is set at build time via ldflags: -X main.version=v1.2.3
// Falls back to "dev" when built without ldflags (e.g. go build locally).
var version = "dev"

func main() {
	cmd.SetVersion(version)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "✗ %s\n", err)
		os.Exit(1)
	}
}
