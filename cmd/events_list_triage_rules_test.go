package cmd

import (
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventsListTriageRulesCmd(t *testing.T) {
	mockResponse := map[string]any{
		"event_get_triage_rules": map[string]any{
			"rules": []map[string]any{
				{
					"id":              "trule-1",
					"name":            "Service Tier 1: postgres",
					"rule_type":       "scoring",
					"action":          "adjust_score",
					"priority":        100,
					"enabled":         true,
					"match_count":     42,
					"last_matched_at": "2026-06-01T00:00:00Z",
					"is_system_rule":  true,
				},
			},
		},
	}

	output, err := testutil.RunWithSimpleGraphQL(mockResponse, eventsCmd, []string{"events", "list-triage-rules"})
	require.NoError(t, err)

	assert.Contains(t, output, "trule-1")
	assert.Contains(t, output, "Service Tier 1: postgres")
	assert.Contains(t, output, "scoring")
	assert.Contains(t, output, "adjust_score")
	assert.Contains(t, output, "100")
}

func TestEventsListTriageRulesCmd_BadEnabledFlag(t *testing.T) {
	eventsListTriageRulesType = ""
	eventsListTriageRulesEnabled = ""

	args := []string{"events", "list-triage-rules", "--enabled", "garbage"}
	_, err := testutil.RunWithSimpleGraphQL(map[string]any{}, eventsCmd, args)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be true or false")
}
