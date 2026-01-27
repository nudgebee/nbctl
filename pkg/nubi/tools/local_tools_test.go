package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteTool_LocalShell(t *testing.T) {
	// Test echo command
	args := map[string]any{
		"command": "echo hello",
	}
	output, err := ExecuteTool(context.Background(), "local_shell", args)
	require.NoError(t, err)
	assert.Equal(t, "hello\n", output)

	// Test missing command
	_, err = ExecuteTool(context.Background(), "local_shell", map[string]any{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required argument: command")
}

func TestExecuteTool_LocalReadFile(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(filePath, []byte("line 1\nline 2\nline 3\nline 4\nline 5"), 0644)
	require.NoError(t, err)

	// Test reading the whole file
	args := map[string]any{
		"path": filePath,
	}
	output, err := ExecuteTool(context.Background(), "local_read_file", args)
	require.NoError(t, err)
	assert.Equal(t, "line 1\nline 2\nline 3\nline 4\nline 5\n", output)

	// Test pagination: offset 1, limit 2
	args = map[string]any{
		"path":   filePath,
		"offset": float64(1),
		"limit":  float64(2),
	}
	output, err = ExecuteTool(context.Background(), "local_read_file", args)
	require.NoError(t, err)
	assert.Equal(t, "line 2\nline 3\n", output)

	// Test invalid path argument
	_, err = ExecuteTool(context.Background(), "local_read_file", map[string]any{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required argument: path")

	// Test binary file detection
	binPath := filepath.Join(tmpDir, "test.bin")
	err = os.WriteFile(binPath, []byte{0x00, 0x01, 0xFF, 0xFE}, 0644)
	require.NoError(t, err)
	output, err = ExecuteTool(context.Background(), "local_read_file", map[string]any{"path": binPath})
	require.NoError(t, err)
	assert.Contains(t, output, "file appears to be binary")
}

func TestExecuteTool_LocalWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	// Test writing to a nested directory that doesn't exist
	filePath := filepath.Join(tmpDir, "nested", "dir", "write_test.txt")

	// Test writing to file
	args := map[string]any{
		"path":    filePath,
		"content": "new content",
	}
	output, err := ExecuteTool(context.Background(), "local_write_file", args)
	require.NoError(t, err)
	assert.Equal(t, "File written successfully.", output)

	// Verify content
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "new content", string(content))

	// Test append mode
	args = map[string]any{
		"path":    filePath,
		"content": " appended",
		"mode":    "append",
	}
	output, err = ExecuteTool(context.Background(), "local_write_file", args)
	require.NoError(t, err)
	assert.Equal(t, "File written successfully.", output)
	content, _ = os.ReadFile(filePath)
	assert.Equal(t, "new content appended", string(content))

	// Test patch mode
	args = map[string]any{
		"path":    filePath,
		"content": "modified",
		"mode":    "patch",
		"search":  "content",
	}
	output, err = ExecuteTool(context.Background(), "local_write_file", args)
	require.NoError(t, err)
	assert.Equal(t, "File written successfully.", output)
	content, _ = os.ReadFile(filePath)
	assert.Equal(t, "new modified appended", string(content))

	// Test missing arguments
	_, err = ExecuteTool(context.Background(), "local_write_file", map[string]any{"path": filePath})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required argument: content")
}

func TestExecuteTool_LocalSearchFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "search_test.txt")
	err := os.WriteFile(filePath, []byte("hello world\nthis is a test\nfind me if you can"), 0644)
	require.NoError(t, err)

	// Test basic search
	args := map[string]any{
		"pattern": "test",
		"path":    tmpDir,
	}
	output, err := ExecuteTool(context.Background(), "local_search_file", args)
	require.NoError(t, err)
	assert.Contains(t, output, "this is a test")

	// Test pattern not found (grep/rg returns error code 1 usually)
	args["pattern"] = "nonexistent_pattern_xyz"
	output, err = ExecuteTool(context.Background(), "local_search_file", args)
	require.NoError(t, err)
	assert.Contains(t, output, "exit status 1")

	// Test missing pattern
	_, err = ExecuteTool(context.Background(), "local_search_file", map[string]any{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required argument: pattern")
}

func TestExecuteTool_LocalGlob(t *testing.T) {
	tmpDir := t.TempDir()
	files := []string{"a.go", "b.go", "c.txt", "subdir/d.go"}
	for _, f := range files {
		p := filepath.Join(tmpDir, f)
		_ = os.MkdirAll(filepath.Dir(p), 0755)
		_ = os.WriteFile(p, []byte("content"), 0644)
	}

	// Test recursive glob
	args := map[string]any{
		"pattern": "**/*.go",
		"path":    tmpDir,
	}
	output, err := ExecuteTool(context.Background(), "local_glob", args)
	require.NoError(t, err)
	assert.Contains(t, output, "a.go")
	assert.Contains(t, output, "b.go")
	assert.Contains(t, output, "subdir/d.go")
	assert.NotContains(t, output, "c.txt")

	// Test non-recursive glob
	args = map[string]any{
		"pattern": "*.go",
		"path":    tmpDir,
	}
	output, err = ExecuteTool(context.Background(), "local_glob", args)
	require.NoError(t, err)
	assert.Contains(t, output, "a.go")
	assert.Contains(t, output, "b.go")
	assert.NotContains(t, output, "subdir/d.go")

	// Test missing pattern
	_, err = ExecuteTool(context.Background(), "local_glob", map[string]any{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required argument: pattern")
}

func TestExecuteTool_UnknownTool(t *testing.T) {
	_, err := ExecuteTool(context.Background(), "unknown_tool", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

func TestExecuteTool_Timeout(t *testing.T) {
	// Test command that sleeps longer than timeout (mocking short timeout for speed if possible,
	// but here we just test the 30s limit behavior with context)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	args := map[string]any{
		"command": "sleep 1",
	}
	output, err := ExecuteTool(ctx, "local_shell", args)
	require.NoError(t, err)
	assert.Contains(t, output, "timed out")
}

func TestGetLocalToolsJSON(t *testing.T) {
	tools, err := GetLocalToolsJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, tools)
	assert.Len(t, tools, 5)

	// Verify structure
	foundShell := false
	for _, tool := range tools {
		if tool["name"] == "local_shell" {
			foundShell = true
			assert.NotEmpty(t, tool["description"])
			assert.NotEmpty(t, tool["input"])

			// Verify input schema is correct map
			inputSchema, ok := tool["input"].(map[string]any)
			assert.True(t, ok)
			assert.Equal(t, "object", inputSchema["type"])
		}
	}
	assert.True(t, foundShell, "local_shell tool not found")
}

func TestIsMutation(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		expected bool
	}{
		{"local_read_file", map[string]any{"path": "foo"}, false},
		{"local_write_file", map[string]any{"path": "foo", "content": "bar"}, true},
		{"local_shell", map[string]any{"command": "ls -la"}, false},
		{"local_shell", map[string]any{"command": "rm -rf /"}, true},
		{"local_shell", map[string]any{"command": "mkdir test"}, true},
		{"local_shell", map[string]any{"command": "echo 'hi' > file.txt"}, true},
		{"local_shell", map[string]any{"command": "cat file.txt"}, false},
		{"local_shell", map[string]any{"command": "npm install pkg"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_"+fmt.Sprint(tt.args["command"]), func(t *testing.T) {
			assert.Equal(t, tt.expected, IsMutation(tt.name, tt.args))
		})
	}
}
