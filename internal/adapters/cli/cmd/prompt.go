package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

func promptLine(label string) (string, error) {
	fmt.Fprintf(os.Stderr, "%s: ", label)

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}

	return strings.TrimSpace(line), nil
}

// promptPassword reads without echo when stdin is a terminal and falls back
// to a plain line read otherwise (pipes, tests).
func promptPassword(label string) (string, error) {
	fd := int(os.Stdin.Fd())

	if !term.IsTerminal(fd) {
		return promptLine(label)
	}

	fmt.Fprintf(os.Stderr, "%s: ", label)
	raw, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}

	return strings.TrimSpace(string(raw)), nil
}
