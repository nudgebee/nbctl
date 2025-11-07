package tools

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/mitchellh/mapstructure"
)

// ShellArgs are the arguments for the ShellTool.
type ShellArgs struct {
	Command string `json:"command"`
}

// ShellTool is a tool that executes a a shell command.
type ShellTool struct{}

// Name returns the name of the tool.
func (t *ShellTool) Name() string {
	return "shell"
}

// Description returns a description of the tool.
func (t *ShellTool) Description() string {
	return "Executes a shell command."
}

// Run executes the tool with the given arguments.
func (t *ShellTool) Run(ctx context.Context, args any) (string, error) {
	var shellArgs ShellArgs
	if err := mapstructure.Decode(args, &shellArgs); err != nil {
		return "", fmt.Errorf("invalid arguments for shell tool: %w", err)
	}

	if shellArgs.Command == "" {
		return "", nil
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", shellArgs.Command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(output), nil
}
