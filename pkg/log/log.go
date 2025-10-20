package log

import (
	"fmt"
	"os"
)

// Errorf prints a formatted error message to stderr.
func Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
}

// InitLogger initializes the logger.
func InitLogger() {
	// Do nothing for now
}
