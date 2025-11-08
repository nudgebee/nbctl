package tools

import (
	"testing"
)

func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	// Test getting a tool that exists
	tool, ok := registry.GetTool("shell")
	if !ok {
		t.Errorf("expected to find tool 'shell'")
	}
	if tool.Name() != "shell" {
		t.Errorf("expected tool name to be 'shell', got %q", tool.Name())
	}

	// Test getting a tool that doesn't exist
	_, ok = registry.GetTool("nonexistent")
	if ok {
		t.Errorf("expected not to find tool 'nonexistent'")
	}
}
