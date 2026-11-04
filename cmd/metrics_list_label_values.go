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

// MetricsListLabelValuesQuery lists the values of a metric label.
const MetricsListLabelValuesQuery = `query MetricsLabelValueList($accountId: String!, $labelName: String!) {
  metrics_list_label_values(request: {account_id: $accountId, label: $labelName}) {
    value
  }
}`

// MetricsListLabelValuesScopedQuery lists a label's values only from series
// matching --match, within the window when one is given.
const MetricsListLabelValuesScopedQuery = `query MetricsLabelValueList($request: FetchMetricsLabelValueRequest!) {
  metrics_list_label_values(request: $request) {
    value
  }
}`

var metricsListLabelValuesCmd = &cobra.Command{
	Use:   "list-label-values",
	Short: "List metric label values",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		accountId, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}

		label, _ := cmd.Flags().GetString("label")
		matchArgs, _ := cmd.Flags().GetStringArray("match")
		matchers, err := parseLabelMatchers(matchArgs)
		if err != nil {
			return err
		}
		startTimeStr, _ := cmd.Flags().GetString("start-time")
		endTimeStr, _ := cmd.Flags().GetString("end-time")

		req := client.NewRequest(MetricsListLabelValuesQuery)
		if len(matchers) == 0 {
			if startTimeStr != "" || endTimeStr != "" {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Warning: --start-time and --end-time apply only with --match; listing the label's values for all time.")
			}
			req.Var("accountId", accountId)
			req.Var("labelName", label)
		} else {
			request := map[string]any{"account_id": accountId, "label": label}
			for _, bound := range []struct{ flag, value, key string }{
				{"--start-time", startTimeStr, "start_time"},
				{"--end-time", endTimeStr, "end_time"},
			} {
				if bound.value == "" {
					continue
				}
				t, err := time.Parse(time.RFC3339, bound.value)
				if err != nil {
					return fmt.Errorf("%s: %w", bound.flag, err)
				}
				request[bound.key] = t.UnixMilli()
			}
			list := make([]map[string]any, 0, len(matchers))
			for _, m := range matchers {
				list = append(list, map[string]any{"label": m.Label, "operator": m.Operator, "value": m.Value})
			}
			request["request"] = map[string]any{"label_matchers": list}
			req = client.NewRequest(MetricsListLabelValuesScopedQuery)
			req.Var("request", request)
		}

		var respData struct {
			MetricsListLabelValues []struct {
				Value string `json:"value"`
			} `json:"metrics_list_label_values"`
		}

		err = graphqlClient.Run(context.Background(), req, &respData)
		if err != nil && len(matchers) > 0 && strings.Contains(err.Error(), matchRefusedMetrics) {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), matchRefusedWarning)
			req = client.NewRequest(MetricsListLabelValuesQuery)
			req.Var("accountId", accountId)
			req.Var("labelName", label)
			err = graphqlClient.Run(context.Background(), req, &respData)
		}
		if err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.MetricsListLabelValues,
			Fields: []format.TableField{
				{Header: "Value", Field: "Value"},
			},
		}
		return printRows(cmd, table, len(respData.MetricsListLabelValues), fmt.Sprintf("No values found for label %q.", label))
	},
}

func init() {
	metricsCmd.AddCommand(metricsListLabelValuesCmd)
	metricsListLabelValuesCmd.Flags().String("account-id", "", "Account ID")
	metricsListLabelValuesCmd.Flags().String("label", "", "Label name; __name__ lists metric names")
	metricsListLabelValuesCmd.Flags().StringArray("match", nil, "Only values from series that also carry this label: label=value, label!=value or label=~regex (repeatable). With --label __name__ this lists the metric names those series have")
	metricsListLabelValuesCmd.Flags().String("start-time", "", "With --match: only values seen from this time (RFC3339)")
	metricsListLabelValuesCmd.Flags().String("end-time", "", "With --match: only values seen until this time (RFC3339)")
	if err := metricsListLabelValuesCmd.MarkFlagRequired("label"); err != nil {
		panic(err)
	}
}
