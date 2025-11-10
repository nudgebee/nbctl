package log

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorf(t *testing.T) {
	// Keep track of the original stderr
	oldStderr := os.Stderr
	// Create a new pipe
	r, w, _ := os.Pipe()
	// Set stderr to the write end of the pipe
	os.Stderr = w

	// Run the function that writes to stderr
	Errorf("test error %s", "message")

	// Close the write end of the pipe
	require.NoError(t, w.Close())
	// Read everything from the read end of the pipe
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	require.NoError(t, err)
	// Restore the original stderr
	os.Stderr = oldStderr

	// Check if the output is what we expected
	assert.Equal(t, "test error message", buf.String())
}
