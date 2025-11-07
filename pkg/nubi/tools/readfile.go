package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/mitchellh/mapstructure"
)

// ReadFileArgs are the arguments for the ReadFileTool.
type ReadFileArgs struct {
	Filepath string `json:"filepath"`
}

// ReadFileTool is a tool that reads the contents of a single file.
type ReadFileTool struct{}

// Name returns the name of the tool.
func (t *ReadFileTool) Name() string {
	return "readfile"
}

// Description returns a description of the tool.
func (t *ReadFileTool) Description() string {
	return "Reads the contents of a single file."
}

// Run executes the tool with the given arguments.
func (t *ReadFileTool) Run(ctx context.Context, args any) (string, error) {
	var readArgs ReadFileArgs
	if err := mapstructure.Decode(args, &readArgs); err != nil {
		return "", fmt.Errorf("invalid arguments for readfile tool: %w", err)
	}

	if readArgs.Filepath == "" {
		return "Usage: readfile <file>", nil
	}

	data, err := os.ReadFile(readArgs.Filepath)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
