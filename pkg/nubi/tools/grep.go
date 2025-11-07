package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/mitchellh/mapstructure"
)

// GrepArgs are the arguments for the GrepTool.
type GrepArgs struct {
	Pattern  string `json:"pattern"`
	Filepath string `json:"filepath"`
}

// GrepTool is a tool that searches for a pattern in a file.
type GrepTool struct{}

// Name returns the name of the tool.
func (t *GrepTool) Name() string {
	return "grep"
}

// Description returns a description of the tool.
func (t *GrepTool) Description() string {
	return "Searches for a pattern in a file."
}

// Run executes the tool with the given arguments.
func (t *GrepTool) Run(ctx context.Context, args any) (string, error) {
	var grepArgs GrepArgs
	if err := mapstructure.Decode(args, &grepArgs); err != nil {
		return "", fmt.Errorf("invalid arguments for grep tool: %w", err)
	}

	if grepArgs.Pattern == "" || grepArgs.Filepath == "" {
		return "Usage: grep <pattern> <file>", nil
	}

	pattern, err := regexp.Compile(grepArgs.Pattern)
	if err != nil {
		return "", err
	}

	file, err := os.Open(grepArgs.Filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var matches []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if pattern.MatchString(line) {
			matches = append(matches, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(matches, "\n"), nil
}
