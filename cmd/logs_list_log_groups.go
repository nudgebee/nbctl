package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/machinebox/graphql"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
)

type LogGroupOutput struct {
	Sample         string `json:"sample"`
	Namespace      string `json:"namespace"`
	Pod            string `json:"pod"`
	Count          string `json:"count"`
	LastOccurrence string `json:"last_occurrence"`
}

var logsListLogGroupsCmd = &cobra.Command{
	Use:   "list-log-groups",
	Short: "List log groups with critical/error messages",
	Long:  `Executes a metrics query to find log groups with critical or error messages and displays them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := client.NewClient()

		accountId, _ := cmd.Flags().GetString("account-id")
		if accountId == "" {
			accountId = viper.GetString("account-id")
		}
		if accountId == "" {
			return fmt.Errorf("account-id is required")
		}

		namespace, _ := cmd.Flags().GetString("namespace")

		// The metrics query to execute
		queryFilter := `level=~"critical|error", container_id!~".*(prometheus|grafana|kube-system|nudgebee-agent|containerd|kubelet).*"`
		if namespace != "" {
			queryFilter = fmt.Sprintf(`%s, container_id=~"/k8s/%s/.*"`, queryFilter, namespace)
		}
		promQLQuery := fmt.Sprintf(`increase(container_log_messages_total{ __CLUSTER__ %s}[1h]) > 0`, queryFilter)

		queries := map[string]any{
			"query": promQLQuery,
		}

		// Set time range for the query (last 1 hour)
		endTime := time.Now()
		startTime := endTime.Add(-1 * time.Hour)

		// Convert to Unix milliseconds
		startTimeMs := float64(startTime.UnixNano() / int64(time.Millisecond))
		endTimeMs := float64(endTime.UnixNano() / int64(time.Millisecond))

		req := graphql.NewRequest(`
			query MetricsQuery(
				$account_id: String!
				$queries: jsonb!
				$instant: Boolean!
				$start_time: Float!
				$end_time: Float!
			) {
				metrics_query(
					request: {
						account_id: $account_id
						queries: $queries
						instant: $instant
						start_time: $start_time
						end_time: $end_time
					}
				) {
					results
				}
			}
		`)

		req.Var("account_id", accountId)
		req.Var("queries", queries) // Pass queries directly
		req.Var("instant", true)
		req.Var("start_time", startTimeMs)
		req.Var("end_time", endTimeMs)

		var rawResp json.RawMessage
		if err := client.Run(context.Background(), req, &rawResp); err != nil {
			return err
		}

		var respData MetricsQueryResponse
		if err := json.Unmarshal(rawResp, &respData); err != nil {
			return fmt.Errorf("failed to unmarshal raw response into MetricsQueryResponse: %w", err)
		}

		// Assuming we only care about the first response in the array
		if len(respData.MetricsQuery.Results) == 0 {
			fmt.Println("No log groups found with critical/error messages.")
			return nil
		}
		metricsResponse := respData.MetricsQuery.Results[0] // Get the first response

		var outputData []LogGroupOutput
		for _, res := range metricsResponse.Payload {
			// Extract sample
			sample := res.Metric["sample"]

			// Extract namespace
			namespace := res.Metric["namespace"]

			// Extract pod
			pod := res.Metric["pod"]

			// Extract count
			count := "0"
			if len(res.Values) > 0 {
				count = strconv.Itoa(len(res.Values))
			}

			// Extract last-occurrence
			lastOccurrence := ""
			if len(res.Timestamps) > 0 {
				tm := time.Unix(int64(res.Timestamps[len(res.Timestamps)-1]), 0)
				lastOccurrence = tm.Format(time.RFC3339)
			}

			outputData = append(outputData, LogGroupOutput{
				Sample:         sample,
				Namespace:      namespace,
				Pod:            pod,
				Count:          count,
				LastOccurrence: lastOccurrence,
			})
		}

		if len(outputData) > 0 {
			for i, item := range outputData {
				format.GetFormat().Print(item)
				if i < len(outputData)-1 {
					fmt.Println("---")
				}
			}
		} else {
			fmt.Println("No log groups found with critical/error messages.")
		}

		return nil
	},
}

func init() {
	logsCmd.AddCommand(logsListLogGroupsCmd)
	logsListLogGroupsCmd.Flags().String("account-id", "", "Account ID")
	logsListLogGroupsCmd.Flags().String("namespace", "", "Filter by namespace (case-insensitive like)")
}
