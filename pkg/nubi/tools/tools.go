package tools

import "context"

// Tool is the interface for a tool that can be executed.
type Tool interface {
	// Name returns the name of the tool.
	Name() string
	// Description returns a description of the tool.
	Description() string
	// Run executes the tool with the given arguments.
	Run(ctx context.Context, args any) (string, error)
}
