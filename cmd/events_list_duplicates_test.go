package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventsListDuplicatesCmd(t *testing.T) {
	mockResponse := map[string]any{
		"event_get_duplicates": map[string]any{
			"duplicates": []map[string]any{
				{
					"event_id":          "evt-2",
					"fingerprint":       "fp-abc",
					"occurrence_number": 2,
					"first_event_id":    "evt-1",
					"starts_at":         "2026-06-01T00:00:00Z",
				},
				{
					"event_id":          "evt-3",
					"fingerprint":       "fp-abc",
					"occurrence_number": 3,
					"first_event_id":    "evt-1",
					"starts_at":         "2026-06-01T00:05:00Z",
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, eventsCmd, []string{"events", "list-duplicates", "evt-1"})
	require.NoError(t, err)

	assert.Contains(t, output, "evt-2")
	assert.Contains(t, output, "evt-3")
	assert.Contains(t, output, "fp-abc")
}

func TestEventsListDuplicatesCmd_Empty(t *testing.T) {
	mockResponse := map[string]any{
		"event_get_duplicates": map[string]any{
			"duplicates": []map[string]any{},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, eventsCmd, []string{"events", "list-duplicates", "evt-1"})
	require.NoError(t, err)
	assert.Contains(t, output, "No duplicates found")
}
