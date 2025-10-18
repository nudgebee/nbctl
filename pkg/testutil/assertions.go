package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// RequireNotEmpty fails the test if s is empty.
func RequireNotEmpty(t testing.TB, s string) {
	require.NotEmpty(t, s)
}

// RequireNoError fails the test if err is non-nil.
func RequireNoError(t testing.TB, err error) {
	require.NoError(t, err)
}
