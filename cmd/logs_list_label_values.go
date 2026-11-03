package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

// LogsListLabelValuesQuery lists the values of a log label in a time window ($query is "start=<ns>&end=<ns>").
const LogsListLabelValuesQuery = `query FetchLogLabelValues($accountId: String!, $labelName: String!, $query: String!) {
  logs_list_label_values(request: {account_id: $accountId, label_name: $labelName, request: {query: $query}}) {
    value
  }
}`

// LogsListLabelValuesWithIndexQuery is LogsListLabelValuesQuery with an index,
// for providers that read values from an index (Elasticsearch).
const LogsListLabelValuesWithIndexQuery = `query FetchLogLabelValues($accountId: String!, $labelName: String!, $query: String!, $index: String!) {
  logs_list_label_values(request: {account_id: $accountId, label_name: $labelName, request: {index: $index, query: $query}}) {
    value
  }
}`

// LogsListLabelValuesScopedQuery lists a label's values only from streams that
// carry the --match labels.
const LogsListLabelValuesScopedQuery = `query FetchLogLabelValues($request: FetchLogLabelValuesRequest!) {
  logs_list_label_values(request: $request) {
    value
  }
}`

var logsListLabelValuesCmd = &cobra.Command{
	Use:   "list-label-values",
	Short: "List log label values",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		accountId, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}

		startTimeStr, _ := cmd.Flags().GetString("start-time")
		endTimeStr, _ := cmd.Flags().GetString("end-time")
		labelName, _ := cmd.Flags().GetString("label-name")

		var startTime, endTime time.Time

		if startTimeStr == "" {
			startTime = time.Now().Add(-1 * time.Hour)
		} else {
			startTime, err = time.Parse(time.RFC3339, startTimeStr)
			if err != nil {
				return err
			}
		}

		if endTimeStr == "" {
			endTime = time.Now()
		} else {
			endTime, err = time.Parse(time.RFC3339, endTimeStr)
			if err != nil {
				return err
			}
		}

		query := fmt.Sprintf("start=%d&end=%d", startTime.UnixNano(), endTime.UnixNano())

		matchArgs, _ := cmd.Flags().GetStringArray("match")
		matchers, err := parseLabelMatchers(matchArgs)
		if err != nil {
			return err
		}
		index, _ := cmd.Flags().GetString("index")

		unscopedRequest := func() *client.Request {
			req := client.NewRequest(LogsListLabelValuesQuery)
			if index != "" {
				req = client.NewRequest(LogsListLabelValuesWithIndexQuery)
				req.Var("index", index)
			}
			req.Var("accountId", accountId)
			req.Var("labelName", labelName)
			req.Var("query", query)
			return req
		}

		req := unscopedRequest()
		if len(matchers) > 0 {
			filters := map[string]string{}
			for _, m := range matchers {
				if m.Operator != "_eq" {
					return fmt.Errorf("--match %s%s%s: logs take label=value only", m.Label, map[string]string{"_neq": "!=", "_regex": "=~"}[m.Operator], m.Value)
				}
				if _, dup := filters[m.Label]; dup {
					return fmt.Errorf("--match %s: a label can be matched once for logs", m.Label)
				}
				filters[m.Label] = m.Value
			}
			params := map[string]any{"query": query}
			if index != "" {
				params["index"] = index
			}
			req = client.NewRequest(LogsListLabelValuesScopedQuery)
			req.Var("request", map[string]any{
				"account_id": accountId, "label_name": labelName, "request": params, "current_filters": filters,
			})
		}

		var respData struct {
			LogsListLabelValues []struct {
				Value string `json:"value"`
			} `json:"logs_list_label_values"`
		}

		err = graphqlClient.Run(context.Background(), req, &respData)
		if err != nil && len(matchers) > 0 && strings.Contains(err.Error(), matchRefusedLogs) {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), matchRefusedWarning)
			err = graphqlClient.Run(context.Background(), unscopedRequest(), &respData)
		}
		if err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.LogsListLabelValues,
			Fields: []format.TableField{
				{Header: "Value", Field: "Value"},
			},
		}
		return printRows(cmd, table, len(respData.LogsListLabelValues), fmt.Sprintf("No values found for log label %q between %s and %s (it may be a field inside log lines rather than an indexed label).", labelName, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339)))
	},
}

func init() {
	logsCmd.AddCommand(logsListLabelValuesCmd)
	logsListLabelValuesCmd.Flags().String("account-id", "", "Account ID")
	logsListLabelValuesCmd.Flags().String("start-time", "", "Start time (RFC3339)")
	logsListLabelValuesCmd.Flags().String("end-time", "", "End time (RFC3339)")
	logsListLabelValuesCmd.Flags().String("label-name", "", "Label name")
	logsListLabelValuesCmd.Flags().StringArray("match", nil, "Only values from log streams that also carry this label: label=value (repeatable), e.g. --label-name pod --match namespace=demo")
	logsListLabelValuesCmd.Flags().String("index", "", "Elasticsearch/OpenSearch only: index to read values from (other providers ignore it)")
	if err := logsListLabelValuesCmd.MarkFlagRequired("label-name"); err != nil {
		panic(err)
	}
}
