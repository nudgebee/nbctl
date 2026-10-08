package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/nudgebee/nbctl/pkg/testutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturedRequest struct {
	Path      string
	Auth      string
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

// resetFlags puts every flag of the command at args back to its default; the
// shared rootCmd keeps flag values from earlier runs otherwise.
func resetFlags(t *testing.T, args []string) {
	t.Helper()
	c, _, err := rootCmd.Find(args)
	require.NoError(t, err)
	c.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// runCapturing runs args against a mock API that answers with data and
// returns the command output and every request nbctl sent.
func runCapturing(t *testing.T, data any, args ...string) (string, []capturedRequest) {
	t.Helper()
	resetFlags(t, args)
	var reqs []capturedRequest
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var c capturedRequest
		_ = json.NewDecoder(r.Body).Decode(&c)
		c.Path = r.URL.Path
		c.Auth = r.Header.Get("Authorization")
		reqs = append(reqs, c)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	})
	out, err := testutil.RunWithMockServer(handler, map[string]any{
		"api-key":    "sk-nb-test",
		"account-id": "acc-1",
	}, rootCmd, args)
	require.NoError(t, err, out)
	return out, reqs
}

// The workspace proxy in llm-server (nudgebee-enterprise#40619) allowlists
// these documents and refuses any request field beyond these, per action:
//
//	metrics_list_names:        account_id
//	metrics_list_labels:       account_id, metric
//	metrics_list_label_values: account_id, label
//	metrics_list:              account_id, queries{query}, instant, start_time, end_time, step_interval
//	logs_list_labels:          account_id, request{query}
//	logs_list_label_values:    account_id, label_name, request{query}
//	logs_list:                 account_id, query, start_time, end_time, limit, offset
//
// Adding or renaming a field here needs the same change in the proxy.
func TestWorkspaceCommandsGraphQLContract(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		data     any
		query    string
		wantVars func(t *testing.T, vars map[string]any)
	}{
		{
			name:  "metrics list-metrics",
			args:  []string{"metrics", "list-metrics"},
			data:  map[string]any{"metrics_list_names": []any{map[string]any{"metric": "up"}}},
			query: MetricsListNamesQuery,
			wantVars: func(t *testing.T, v map[string]any) {
				assert.Equal(t, map[string]any{"accountId": "acc-1"}, v)
			},
		},
		{
			name:  "metrics list-labels",
			args:  []string{"metrics", "list-labels", "--metric", "up"},
			data:  map[string]any{"metrics_list_labels": []any{}},
			query: MetricsListLabelsQuery,
			wantVars: func(t *testing.T, v map[string]any) {
				assert.Equal(t, map[string]any{"accountId": "acc-1", "metricName": "up"}, v)
			},
		},
		{
			name:  "metrics list-label-values",
			args:  []string{"metrics", "list-label-values", "--label", "job"},
			data:  map[string]any{"metrics_list_label_values": []any{}},
			query: MetricsListLabelValuesQuery,
			wantVars: func(t *testing.T, v map[string]any) {
				assert.Equal(t, map[string]any{"accountId": "acc-1", "labelName": "job"}, v)
			},
		},
		{
			name: "metrics query",
			args: []string{"metrics", "query", "--query", "up", "--step", "5m",
				"--start-time", "2026-10-01T00:00:00Z", "--end-time", "2026-10-01T01:00:00Z"},
			data:  map[string]any{"metrics_list": map[string]any{"results": []any{}}},
			query: MetricsQueryQuery,
			wantVars: func(t *testing.T, v map[string]any) {
				assert.Equal(t, map[string]any{"request": map[string]any{
					"account_id":    "acc-1",
					"queries":       map[string]any{"query": "up"},
					"instant":       false,
					"start_time":    float64(1790812800000),
					"end_time":      float64(1790816400000),
					"step_interval": float64(300),
				}}, v)
			},
		},
		{
			name:  "logs list-labels",
			args:  []string{"logs", "list-labels", "--start-time", "2026-10-01T00:00:00Z", "--end-time", "2026-10-01T01:00:00Z"},
			data:  map[string]any{"logs_list_labels": []any{}},
			query: LogsListLabelsQuery,
			wantVars: func(t *testing.T, v map[string]any) {
				assert.Equal(t, map[string]any{"accountId": "acc-1", "query": "start=1790812800000000000&end=1790816400000000000"}, v)
			},
		},
		{
			name: "logs list-label-values",
			args: []string{"logs", "list-label-values", "--label-name", "app",
				"--start-time", "2026-10-01T00:00:00Z", "--end-time", "2026-10-01T01:00:00Z"},
			data:  map[string]any{"logs_list_label_values": []any{}},
			query: LogsListLabelValuesQuery,
			wantVars: func(t *testing.T, v map[string]any) {
				assert.Equal(t, map[string]any{"accountId": "acc-1", "labelName": "app", "query": "start=1790812800000000000&end=1790816400000000000"}, v)
			},
		},
		{
			name: "logs query",
			args: []string{"logs", "query", "--query", `{app="api"}`, "--limit", "500", "--offset", "100",
				"--start-time", "2026-10-01T00:00:00Z", "--end-time", "2026-10-01T01:00:00Z"},
			data:  map[string]any{"logs_list": map[string]any{"logs": []any{}}},
			query: LogsQueryQuery,
			wantVars: func(t *testing.T, v map[string]any) {
				assert.Equal(t, map[string]any{"request": map[string]any{
					"account_id": "acc-1",
					"query":      `{app="api"}`,
					"start_time": float64(1790812800000),
					"end_time":   float64(1790816400000),
					"limit":      float64(500),
					"offset":     float64(100),
				}}, v)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, reqs := runCapturing(t, tt.data, tt.args...)
			require.Len(t, reqs, 1, "one GraphQL request per command, nothing else")
			assert.Equal(t, "/api/graphql", reqs[0].Path)
			assert.Equal(t, "Bearer sk-nb-test", reqs[0].Auth)
			assert.Equal(t, tt.query, reqs[0].Query)
			tt.wantVars(t, reqs[0].Variables)
		})
	}
}

func TestMetricsQueryOmitsStepByDefault(t *testing.T) {
	_, reqs := runCapturing(t, map[string]any{"metrics_list": map[string]any{"results": []any{}}},
		"metrics", "query", "--query", "up")
	require.Len(t, reqs, 1)
	assert.NotContains(t, reqs[0].Variables["request"], "step_interval")
}

func TestMetricsQueryJSONIsBackendResults(t *testing.T) {
	results := `[{"query_key":"query","query":"up","payload":[{"metric":{"__name__":"up","job":"api"},"timestamps":[1790812800,1790812860],"values":[1,null],"non_finite":{"nan":1}}]}]`
	var data any
	require.NoError(t, json.Unmarshal([]byte(`{"metrics_list":{"results":`+results+`}}`), &data))

	out, _ := runCapturing(t, data, "metrics", "query", "--query", "up", "-o", "json")
	assert.JSONEq(t, results, out)
}

func TestMetricsQueryJSONPassesThroughUnexpectedShapes(t *testing.T) {
	// A label value that is not a string would not decode into MetricsResult;
	// JSON output must still print it unchanged.
	results := `[{"query_key":"query","payload":[{"metric":{"le":0.5,"job":"api"},"timestamps":[1],"values":[2]}]}]`
	var data any
	require.NoError(t, json.Unmarshal([]byte(`{"metrics_list":{"results":`+results+`}}`), &data))

	out, _ := runCapturing(t, data, "metrics", "query", "--query", "up", "-o", "json")
	assert.JSONEq(t, results, out)
}

func TestMetricsQueryJSONEmpty(t *testing.T) {
	out, _ := runCapturing(t, map[string]any{"metrics_list": map[string]any{"results": nil}},
		"metrics", "query", "--query", "up", "-o", "json")
	assert.JSONEq(t, `[]`, out)
}

func TestLogsQueryJSONIsBackendLogs(t *testing.T) {
	logs := `[{"timestamp":"2026-10-01T00:00:01Z","severity":"error","message":"boom","labels":{"app":"api","pod":"api-1"}}]`
	var data any
	require.NoError(t, json.Unmarshal([]byte(`{"logs_list":{"logs":`+logs+`}}`), &data))

	out, _ := runCapturing(t, data, "logs", "query", "--query", `{app="api"}`, "-o", "json")
	assert.JSONEq(t, logs, out)

	out, _ = runCapturing(t, map[string]any{"logs_list": map[string]any{"logs": []any{}}},
		"logs", "query", "--query", `{app="api"}`, "-o", "json")
	assert.JSONEq(t, `[]`, out)
}

func TestRestrictCommands(t *testing.T) {
	root := &cobra.Command{Use: "nbctl"}
	for _, name := range []string{"metrics", "logs", "tickets", "workflow", "version", "completion"} {
		root.AddCommand(&cobra.Command{Use: name})
	}
	names := func() []string {
		var out []string
		for _, c := range root.Commands() {
			out = append(out, c.Name())
		}
		return out
	}

	restrictCommands(root, "")
	assert.Len(t, names(), 6, "an empty list changes nothing")

	restrictCommands(root, " metrics, logs ,")
	assert.ElementsMatch(t, []string{"metrics", "logs", "version", "completion"}, names())
}

// runCapturingStderr is runCapturing that also returns what went to stderr.
func runCapturingStderr(t *testing.T, data any, args ...string) (string, string) {
	t.Helper()
	var errBuf bytes.Buffer
	rootCmd.SetErr(&errBuf)
	defer rootCmd.SetErr(nil)
	out, _ := runCapturing(t, data, args...)
	return out, errBuf.String()
}

func TestEmptyResultsExplainOnStderr(t *testing.T) {
	window := []string{"--start-time", "2026-10-01T00:00:00Z", "--end-time", "2026-10-01T01:00:00Z"}
	tests := []struct {
		name    string
		args    []string
		data    any
		wantErr string
	}{
		{"metrics list-metrics", []string{"metrics", "list-metrics"},
			map[string]any{"metrics_list_names": []any{}}, "No metrics found"},
		{"metrics list-labels", []string{"metrics", "list-labels", "--metric", "up"},
			map[string]any{"metrics_list_labels": nil}, `No labels found for metric "up"`},
		{"metrics list-label-values", []string{"metrics", "list-label-values", "--label", "job"},
			map[string]any{"metrics_list_label_values": []any{}}, `No values found for label "job"`},
		{"logs list-labels", append([]string{"logs", "list-labels"}, window...),
			map[string]any{"logs_list_labels": []any{}}, "No log labels found between 2026-10-01T00:00:00Z and 2026-10-01T01:00:00Z"},
		{"logs list-label-values", append([]string{"logs", "list-label-values", "--label-name", "severity"}, window...),
			map[string]any{"logs_list_label_values": []any{}}, `No values found for log label "severity"`},
		{"logs query", append([]string{"logs", "query", "--query", "x"}, window...),
			map[string]any{"logs_list": map[string]any{"logs": []any{}}}, "No logs found between"},
		{"metrics query", append([]string{"metrics", "query", "--query", "up"}, window...),
			map[string]any{"metrics_list": map[string]any{"results": []any{map[string]any{"query_key": "query", "payload": []any{}}}}}, "No data"},
	}
	for _, tt := range tests {
		t.Run(tt.name+" text", func(t *testing.T) {
			out, stderr := runCapturingStderr(t, tt.data, tt.args...)
			assert.Empty(t, strings.TrimSpace(out))
			assert.Contains(t, stderr, tt.wantErr)
		})
		t.Run(tt.name+" json", func(t *testing.T) {
			out, stderr := runCapturingStderr(t, tt.data, append(tt.args, "-o", "json")...)
			if tt.name == "metrics query" {
				assert.JSONEq(t, `[{"query_key":"query","payload":[]}]`, out)
			} else {
				assert.JSONEq(t, `[]`, out)
			}
			assert.Contains(t, stderr, tt.wantErr)
		})
	}
}

func TestLogsQueryWarnsWhenLimitReached(t *testing.T) {
	entry := map[string]any{"timestamp": "t", "severity": "info", "message": "m", "labels": map[string]any{}}
	data := map[string]any{"logs_list": map[string]any{"logs": []any{entry, entry}}}

	out, stderr := runCapturingStderr(t, data, "logs", "query", "--query", "x", "--limit", "2", "--offset", "4", "-o", "json")
	assert.Contains(t, stderr, "Returned 2 lines = --limit; results are probably cut off")
	assert.Contains(t, stderr, "--offset 6")
	var parsed []any
	require.NoError(t, json.Unmarshal([]byte(out), &parsed), "stdout must stay valid JSON")

	_, stderr = runCapturingStderr(t, data, "logs", "query", "--query", "x", "--limit", "3")
	assert.NotContains(t, stderr, "cut off")
}
