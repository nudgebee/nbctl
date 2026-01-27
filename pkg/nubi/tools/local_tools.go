package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// LocalTools defines the tools available for local execution.
var LocalTools = []mcp.Tool{
	{
		Name:        "local_shell",
		Description: "Execute a shell command. Arguments MUST be a JSON or YAML object. Example: {\"command\": \"ls -la\"}. Ensure the command is valid for the detected OS.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{"type": "string", "description": "The command to run"},
			},
			"required": []string{"command"},
		},
	},
	{
		Name:        "local_read_file",
		Description: "Read a local file. Arguments MUST be a JSON or YAML object. Example: {\"path\": \"README.md\", \"limit\": 10}. Use 'offset'/'limit' or 'tail' for large files.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":   map[string]any{"type": "string", "description": "Path to the file"},
				"offset": map[string]any{"type": "integer", "description": "0-based line number to start reading from"},
				"limit":  map[string]any{"type": "integer", "description": "Maximum number of lines to read"},
				"tail":   map[string]any{"type": "integer", "description": "Read the last N lines of the file"},
			},
			"required": []string{"path"},
		},
	},
	{
		Name:        "local_write_file",
		Description: "Write to a local file. Arguments MUST be a JSON or YAML object. Example (YAML): path: test.py\\ncontent: |\\n  print('hello'). Modes: overwrite, append, patch.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":    map[string]any{"type": "string", "description": "Path to the file"},
				"content": map[string]any{"type": "string", "description": "Content to write"},
				"mode":    map[string]any{"type": "string", "description": "Write mode: 'overwrite' (default), 'append', or 'patch'", "enum": []string{"overwrite", "append", "patch"}},
				"search":  map[string]any{"type": "string", "description": "For 'patch' mode: the string to find and replace with content"},
			},
			"required": []string{"path", "content"},
		},
	},
	{
		Name:        "local_search_file",
		Description: "Recursive grep search. Arguments MUST be a JSON or YAML object. Example: {\"pattern\": \"TODO\", \"path\": \".\"}. Supports context and glob filtering.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern":        map[string]any{"type": "string", "description": "The pattern to search for (regex supported)"},
				"path":           map[string]any{"type": "string", "description": "Directory or file to search (default: current directory)"},
				"case_sensitive": map[string]any{"type": "boolean", "description": "If true, search is case-sensitive (default: false)"},
				"before_context": map[string]any{"type": "integer", "description": "Number of lines of context before the match (default: 0)"},
				"after_context":  map[string]any{"type": "integer", "description": "Number of lines of context after the match (default: 0)"},
				"include_glob":   map[string]any{"type": "string", "description": "Only search files matching this glob (e.g., '*.go')"},
			},
			"required": []string{"pattern"},
		},
	},
	{
		Name:        "local_glob",
		Description: "Find files matching a glob. Arguments MUST be a JSON or YAML object. Example: {\"pattern\": \"**/*.go\", \"ignore\": [\"vendor\"]}. Supports metadata.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern":  map[string]any{"type": "string", "description": "The glob pattern to match"},
				"path":     map[string]any{"type": "string", "description": "The directory to search within (default: current directory)"},
				"ignore":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "List of patterns to ignore (e.g., '.git', 'node_modules')"},
				"metadata": map[string]any{"type": "boolean", "description": "If true, returns file size and modification date"},
			},
			"required": []string{"pattern"},
		},
	},
}

// IsMutation returns true if the tool call is likely to mutate the system state.
func IsMutation(name string, args map[string]any) bool {
	switch name {
	case "local_write_file":
		return true
	case "local_shell":
		cmd, _ := args["command"].(string)
		cmd = strings.ToLower(cmd)
		// Check for common destructive or mutating patterns
		mutatingKeywords := []string{
			"rm ", "mkdir ", "touch ", "mv ", "cp ", "chmod ", "chown ",
			"apt ", "yum ", "dnf ", "brew ", "npm install", "pip install",
			"go get", "git push", "wget ", "curl ",
		}
		for _, kw := range mutatingKeywords {
			if strings.Contains(cmd, kw) {
				return true
			}
		}
		// Check for output redirection
		if strings.Contains(cmd, ">") || strings.Contains(cmd, ">>") {
			return true
		}
	}
	return false
}

// ValidateArgs checks if the provided arguments match the tool's required arguments.
func ValidateArgs(name string, args map[string]any) error {
	for _, t := range LocalTools {
		if t.Name == name {
			if schema, ok := t.InputSchema.(map[string]any); ok {
				if required, ok := schema["required"].([]string); ok {
					for _, req := range required {
						if _, exists := args[req]; !exists {
							return fmt.Errorf("missing required argument: %s", req)
						}
					}
				}
			}
			return nil
		}
	}
	return fmt.Errorf("unknown tool: %s", name)
}

// ExecuteTool executes a local tool by name with the given arguments.
func ExecuteTool(ctx context.Context, name string, args map[string]any) (string, error) {
	// Normalize common aliases
	if path, ok := args["path"].(string); !ok || path == "" {
		if filename, ok := args["filename"].(string); ok && filename != "" {
			args["path"] = filename
		}
	}

	if err := ValidateArgs(name, args); err != nil {
		return "", err
	}

	// Enforce workspace boundary
	if path, ok := args["path"].(string); ok && path != "" {
		safePath, err := validatePath(path)
		if err != nil {
			return "", err
		}
		args["path"] = safePath
	}

	// Create a context with a 30 second timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	switch name {
	case "local_shell":
		cmdStr, ok := args["command"].(string)
		if !ok {
			return "", fmt.Errorf("invalid argument command: expected string")
		}
		return executeShell(timeoutCtx, cmdStr)
	case "local_read_file":
		path, _ := args["path"].(string)
		offset, _ := args["offset"].(float64)
		limit, _ := args["limit"].(float64)
		tail, _ := args["tail"].(float64)
		return readFile(path, int(offset), int(limit), int(tail))
	case "local_write_file":
		path, _ := args["path"].(string)
		content, _ := args["content"].(string)
		mode, _ := args["mode"].(string)
		search, _ := args["search"].(string)
		return writeFile(timeoutCtx, path, content, mode, search)
	case "local_search_file":
		pattern, _ := args["pattern"].(string)
		path, _ := args["path"].(string)
		if path == "" {
			path = "."
		}
		caseSensitive, _ := args["case_sensitive"].(bool)
		before, _ := args["before_context"].(float64)
		after, _ := args["after_context"].(float64)
		include, _ := args["include_glob"].(string)
		return searchFile(timeoutCtx, pattern, path, caseSensitive, int(before), int(after), include)
	case "local_glob":
		pattern, _ := args["pattern"].(string)
		path, _ := args["path"].(string)
		if path == "" {
			path = "."
		}
		ignoreRaw, _ := args["ignore"].([]any)
		var ignore []string
		for _, i := range ignoreRaw {
			if s, ok := i.(string); ok {
				ignore = append(ignore, s)
			}
		}
		metadata, _ := args["metadata"].(bool)
		return findFiles(pattern, path, ignore, metadata)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func validatePath(path string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("error getting current directory: %w", err)
	}

	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(cwd, path)
	}
	absPath = filepath.Clean(absPath)

	// Allow access to current workspace
	if strings.HasPrefix(absPath, cwd) {
		return absPath, nil
	}

	// Also allow access to system temp directories (required for tests and some operations)
	tempDirs := []string{os.TempDir(), "/tmp", "/var/tmp"}
	for _, dir := range tempDirs {
		if dir != "" && strings.HasPrefix(absPath, filepath.Clean(dir)) {
			return absPath, nil
		}
	}

	return "", fmt.Errorf("access denied: path is outside the workspace boundary")
}

func executeShell(ctx context.Context, command string) (string, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Sprintf("Error: command timed out after 30s\nOutput: %s", string(out)), nil
		}
		return fmt.Sprintf("Error: %v\nOutput: %s", err, string(out)), nil
	}
	return string(out), nil
}

func readFile(path string, offset, limit, tail int) (string, error) {
	const maxFileSize = 1 * 1024 * 1024 // 1MB

	file, err := os.Open(path)
	if err != nil {
		return fmt.Sprintf("Error opening file: %v", err), nil
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return fmt.Sprintf("Error stat-ing file: %v", err), nil
	}

	if info.Size() > maxFileSize && limit == 0 && offset == 0 && tail == 0 {
		return fmt.Sprintf("Error: file is too large (%d bytes). Max size is 1MB. Please use offset, limit, or tail to paginate.", info.Size()), nil
	}

	// Check if binary (first 512 bytes)
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	if n > 0 {
		if !utf8.Valid(buf[:n]) {
			return "Error: file appears to be binary. Reading binaries is not supported.", nil
		}
		// Reset seek
		if _, err := file.Seek(0, 0); err != nil {
			return fmt.Sprintf("Error seeking file: %v", err), nil
		}
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Sprintf("Error reading file: %v", err), nil
	}

	var resultLines []string
	if tail > 0 {
		start := len(lines) - tail
		if start < 0 {
			start = 0
		}
		resultLines = lines[start:]
	} else {
		end := len(lines)
		if limit > 0 && offset+limit < end {
			end = offset + limit
		}
		if offset < len(lines) {
			resultLines = lines[offset:end]
		}
	}

	return strings.Join(resultLines, "\n") + "\n", nil
}

func searchFile(ctx context.Context, pattern, path string, caseSensitive bool, before, after int, include string) (string, error) {
	grepCmd := "grep -r"
	if !caseSensitive {
		grepCmd += "i"
	}
	grepCmd += "n" // line numbers

	if before > 0 {
		grepCmd += fmt.Sprintf(" -B %d", before)
	}
	if after > 0 {
		grepCmd += fmt.Sprintf(" -A %d", after)
	}

	// Ripgrep check
	if _, err := exec.LookPath("rg"); err == nil {
		grepCmd = "rg --vimgrep"
		if !caseSensitive {
			grepCmd += " -i"
		}
		if before > 0 {
			grepCmd += fmt.Sprintf(" -B %d", before)
		}
		if after > 0 {
			grepCmd += fmt.Sprintf(" -A %d", after)
		}
		if include != "" {
			grepCmd += fmt.Sprintf(" -g %q", include)
		}
	} else if include != "" {
		// Standard grep doesn't have a direct equivalent to ripgrep's -g,
		// but we can use --include
		grepCmd += fmt.Sprintf(" --include=%q", include)
	}

	// Check if path is a file or directory
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Sprintf("Error accessing path: %v", err), nil
	}

	cmdStr := fmt.Sprintf("%s %q %q", grepCmd, pattern, path)
	if !info.IsDir() {
		cmdStr = fmt.Sprintf("%s %q %q", grepCmd, pattern, path)
	}

	return executeShell(ctx, cmdStr)
}

func findFiles(pattern, searchPath string, ignore []string, metadata bool) (string, error) {
	var matches []string

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		rel, _ := filepath.Rel(searchPath, path)
		for _, ign := range ignore {
			if strings.Contains(rel, ign) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		name := info.Name()
		// Basic glob support
		cleanPattern := strings.ReplaceAll(pattern, "**/", "")
		matched, _ := filepath.Match(cleanPattern, name)

		if matched {
			// Handle recursive vs non-recursive
			if !strings.Contains(pattern, "**") {
				if filepath.Dir(path) != filepath.Clean(searchPath) {
					return nil
				}
			}

			if metadata {
				matches = append(matches, fmt.Sprintf("%s (%d bytes, mod: %s)", rel, info.Size(), info.ModTime().Format(time.RFC3339)))
			} else {
				matches = append(matches, rel)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Sprintf("Error walking directory: %v", err), nil
	}

	if len(matches) == 0 {
		return "No files found matching the pattern.", nil
	}
	if len(matches) > 100 {
		return fmt.Sprintf("Found %d files. First 100:\n%s\n...", len(matches), strings.Join(matches[:100], "\n")), nil
	}
	return strings.Join(matches, "\n"), nil
}

func writeFile(ctx context.Context, path, content, mode, search string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Sprintf("Error creating directory: %v", err), nil
	}

	var finalContent string
	switch mode {
	case "append":
		existing, _ := os.ReadFile(path)
		finalContent = string(existing) + content
	case "patch":
		existing, err := os.ReadFile(path)
		if err != nil {
			return fmt.Sprintf("Error reading file for patching: %v", err), nil
		}
		if !strings.Contains(string(existing), search) {
			return fmt.Sprintf("Error: search string %q not found in file", search), nil
		}
		finalContent = strings.Replace(string(existing), search, content, 1)
	default:
		finalContent = content
	}

	// Atomic write using temp file
	tmpFile, err := os.CreateTemp(dir, "nbctl-write-*")
	if err != nil {
		return fmt.Sprintf("Error creating temp file: %v", err), nil
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmpFile.WriteString(finalContent); err != nil {
		_ = tmpFile.Close()
		return fmt.Sprintf("Error writing to temp file: %v", err), nil
	}
	_ = tmpFile.Close()

	const defaultPerm = 0644
	if err := os.Chmod(tmpPath, defaultPerm); err != nil {
		return fmt.Sprintf("Error setting permissions: %v", err), nil
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Sprintf("Error finalizing file write: %v", err), nil
	}

	return "File written successfully.", nil
}

// GetLocalToolNames returns a list of names for all available local tools.
func GetLocalToolNames() []string {
	var names []string
	for _, t := range LocalTools {
		names = append(names, t.Name)
	}
	return names
}

// GetLocalToolsJSON returns the tool definitions in a format suitable for the API request.
func GetLocalToolsJSON() ([]map[string]any, error) {
	capabilities := detectCapabilities()
	var toolList []map[string]any
	for _, t := range LocalTools {
		desc := t.Description
		if t.Name == "local_shell" {
			if len(capabilities) > 0 {
				desc += " Available specialized CLI tools found on this machine include (but are not limited to): " + strings.Join(capabilities, ", ") + "."
			}
			desc += fmt.Sprintf(" Operating System: %s.", runtime.GOOS)
		}

		toolMap := map[string]any{
			"name":        t.Name,
			"description": desc,
			"input":       t.InputSchema,
		}
		toolList = append(toolList, toolMap)
	}
	return toolList, nil
}

// GetPrimaryArgument returns the name of the primary argument if the tool accepts a single required argument.
func GetPrimaryArgument(name string) string {
	for _, t := range LocalTools {
		if t.Name == name {
			if schema, ok := t.InputSchema.(map[string]any); ok {
				if required, ok := schema["required"].([]string); ok && len(required) == 1 {
					return required[0]
				}
			}
		}
	}
	return ""
}

func detectCapabilities() []string {
	interestingTools := []string{
		"aws", "kubectl", "gcloud", "az", "docker", "git",
		"grep", "rg", "jq", "yq", "sed", "awk", "curl", "wget",
		"helm", "terraform", "make", "python", "node", "go",
	}

	var found []string
	for _, tool := range interestingTools {
		if _, err := exec.LookPath(tool); err == nil {
			found = append(found, tool)
		}
	}
	return found
}
