package tools

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestShellTool(t *testing.T) {
	tool := &ShellTool{}
	ctx := context.Background()

	// Test with a simple command
	output, err := tool.Run(ctx, ShellArgs{Command: "echo hello"})
	if err != nil {
		t.Fatalf("ShellTool.Run() error = %v", err)
	}
	if strings.TrimSpace(output) != "hello" {
		t.Errorf("ShellTool.Run() = %q, want %q", output, "hello")
	}

	// Test with a command with quoted arguments
	output, err = tool.Run(ctx, ShellArgs{Command: "echo 'hello world'"})
	if err != nil {
		t.Fatalf("ShellTool.Run() error = %v", err)
	}
	if strings.TrimSpace(output) != "hello world" {
		t.Errorf("ShellTool.Run() = %q, want %q", output, "hello world")
	}

	// Test with no arguments
	output, err = tool.Run(ctx, ShellArgs{})
	if err != nil {
		t.Fatalf("ShellTool.Run() error = %v", err)
	}
	if output != "" {
		t.Errorf("ShellTool.Run() = %q, want %q", output, "")
	}
}

func TestGrepTool(t *testing.T) {
	tool := &GrepTool{}
	ctx := context.Background()

	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := "hello world\nhello grep"
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Test with a matching pattern
	output, err := tool.Run(ctx, GrepArgs{Pattern: "hello", Filepath: tmpfile.Name()})
	if err != nil {
		t.Fatalf("GrepTool.Run() error = %v", err)
	}
	expected := "hello world\nhello grep"
	if strings.TrimSpace(output) != expected {
		t.Errorf("GrepTool.Run() = %q, want %q", output, expected)
	}

	// Test with a non-matching pattern
	output, err = tool.Run(ctx, GrepArgs{Pattern: "goodbye", Filepath: tmpfile.Name()})
	if err != nil {
		t.Fatalf("GrepTool.Run() error = %v", err)
	}
	if output != "" {
		t.Errorf("GrepTool.Run() = %q, want %q", output, "")
	}
}

func TestReadFileTool(t *testing.T) {
	tool := &ReadFileTool{}
	ctx := context.Background()

	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := "hello world"
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Test reading the file
	output, err := tool.Run(ctx, ReadFileArgs{Filepath: tmpfile.Name()})
	if err != nil {
		t.Fatalf("ReadFileTool.Run() error = %v", err)
	}
	if output != content {
		t.Errorf("ReadFileTool.Run() = %q, want %q", output, content)
	}
}

func TestReadManyFilesTool(t *testing.T) {
	tool := &ReadManyFilesTool{}
	ctx := context.Background()

	// Create temporary files
	tmpfile1, err := os.CreateTemp("", "test1")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile1.Name())
	content1 := "hello file1"
	if _, err := tmpfile1.Write([]byte(content1)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile1.Close(); err != nil {
		t.Fatal(err)
	}

	tmpfile2, err := os.CreateTemp("", "test2")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile2.Name())
	content2 := "hello file2"
	if _, err := tmpfile2.Write([]byte(content2)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile2.Close(); err != nil {
		t.Fatal(err)
	}

	// Test reading the files
	output, err := tool.Run(ctx, ReadManyFilesArgs{Filepaths: []string{tmpfile1.Name(), tmpfile2.Name()}})
	if err != nil {
		t.Fatalf("ReadManyFilesTool.Run() error = %v", err)
	}
	expected := "--- " + tmpfile1.Name() + " ---\n" + content1 + "\n--- " + tmpfile2.Name() + " ---\n" + content2 + "\n"
	if output != expected {
		t.Errorf("ReadManyFilesTool.Run() = %q, want %q", output, expected)
	}
}

func TestSearchTool(t *testing.T) {
	tool := &SearchTool{}
	ctx := context.Background()

	// Create a temporary directory and files
	tmpdir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	// Change to the temporary directory
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpdir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)

	// Create files
	if err := os.WriteFile("file1.txt", []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("file2.txt", []byte("hello search"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir("subdir", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("subdir/file3.txt", []byte("another file"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test with a matching pattern
	output, err := tool.Run(ctx, SearchArgs{Pattern: "hello"})
	if err != nil {
		t.Fatalf("SearchTool.Run() error = %v", err)
	}
	if !strings.Contains(output, "file1.txt") || !strings.Contains(output, "file2.txt") {
		t.Errorf("SearchTool.Run() = %q, want it to contain file1.txt and file2.txt", output)
	}

	// Test with a non-matching pattern
	output, err = tool.Run(ctx, SearchArgs{Pattern: "goodbye"})
	if err != nil {
		t.Fatalf("SearchTool.Run() error = %v", err)
	}
	if output != "No matches found." {
		t.Errorf("SearchTool.Run() = %q, want %q", output, "No matches found.")
	}
}
