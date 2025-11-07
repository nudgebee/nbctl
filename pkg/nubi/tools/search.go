package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"
)

// SearchArgs are the arguments for the SearchTool.
type SearchArgs struct {
	Pattern string `json:"pattern"`
}

// SearchTool is a tool that searches for a pattern in files recursively.
type SearchTool struct{}

// Name returns the name of the tool.
func (t *SearchTool) Name() string {
	return "search"
}

// Description returns a description of the tool.
func (t *SearchTool) Description() string {
	return "Searches for a pattern in files recursively in the current directory."
}

// Run executes the tool with the given arguments.
func (t *SearchTool) Run(ctx context.Context, args any) (string, error) {
	var searchArgs SearchArgs
	if err := mapstructure.Decode(args, &searchArgs); err != nil {
		return "", fmt.Errorf("invalid arguments for search tool: %w", err)
	}

	if searchArgs.Pattern == "" {
		return "Usage: search <pattern>", nil
	}
	pattern := searchArgs.Pattern
	var results []string

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// Read file content
			content, err := os.ReadFile(path)
			if err != nil {
				// Log errors for files that can't be read
				fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", path, err)
				return nil
			}
			if strings.Contains(string(content), pattern) {
				results = append(results, path)
			}
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "No matches found.", nil
	}

	return strings.Join(results, "\n"), nil
}
