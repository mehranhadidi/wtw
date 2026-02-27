// Package ui handles terminal output and user input.
// SetReader lets tests inject responses without touching os.Stdin.
package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[0;33m"
	colorRed    = "\033[0;31m"
	colorReset  = "\033[0m"
)

var reader = bufio.NewReader(os.Stdin)

// SetReader replaces the input reader (used in tests to inject canned responses).
func SetReader(r io.Reader) {
	reader = bufio.NewReader(r)
}

// Success prints a green success message.
func Success(msg string) { fmt.Printf("%s✓ %s%s\n", colorGreen, msg, colorReset) }

// Error prints a red error message to stderr.
func Error(msg string) { fmt.Fprintf(os.Stderr, "%s✗ %s%s\n", colorRed, msg, colorReset) }

// PrintCmd prints a green command hint.
func PrintCmd(msg string) { fmt.Printf("%s%s%s\n", colorGreen, msg, colorReset) }

// Confirm prints prompt and returns true if the user answers y/Y (or presses
// enter when def is "Y").
func Confirm(prompt, def string) bool {
	fmt.Printf("%s%s %s", colorYellow, prompt, colorReset)
	line, _ := reader.ReadString('\n')
	reply := strings.TrimSpace(line)
	if reply == "" {
		return def == "Y"
	}
	return reply == "y" || reply == "Y"
}

// Ask prints prompt and returns the trimmed response.
func Ask(prompt string) string {
	fmt.Printf("%s%s %s", colorYellow, prompt, colorReset)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}
