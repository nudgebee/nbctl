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

// Registry is a tool registry.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: map[string]Tool{
			"shell":         &ShellTool{},
			"grep":          &GrepTool{},
			"readfile":      &ReadFileTool{},
			"readmanyfiles": &ReadManyFilesTool{},
			"search":        &SearchTool{},
		},
	}
}

// GetTool returns a tool from the registry.
func (r *Registry) GetTool(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}
