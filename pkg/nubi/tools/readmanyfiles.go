package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/mapstructure"
)

// ReadManyFilesArgs are the arguments for the ReadManyFilesTool.
type ReadManyFilesArgs struct {
	Filepaths []string `json:"filepaths"`
}

// ReadManyFilesTool is a tool that reads the contents of multiple files.
type ReadManyFilesTool struct{}

// Name returns the name of the tool.
func (t *ReadManyFilesTool) Name() string {
	return "readmanyfiles"
}

// Description returns a description of the tool.
func (t *ReadManyFilesTool) Description() string {
	return "Reads the contents of multiple files."
}

// Run executes the tool with the given arguments.
func (t *ReadManyFilesTool) Run(ctx context.Context, args any) (string, error) {
	var readArgs ReadManyFilesArgs
	if err := mapstructure.Decode(args, &readArgs); err != nil {
		return "", fmt.Errorf("invalid arguments for readmanyfiles tool: %w", err)
	}

	if len(readArgs.Filepaths) == 0 {
		return "Usage: readmanyfiles <file1> <file2> ...", nil
	}

	var builder strings.Builder
	for _, file := range readArgs.Filepaths {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		builder.WriteString(fmt.Sprintf("--- %s ---\n%s\n", file, string(data)))
	}

	return builder.String(), nil
}
