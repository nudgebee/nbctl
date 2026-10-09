package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

// LogsListLabelsQuery lists log labels. The nested request carries the time
// window ("start=<ns>&end=<ns>") and, with --index, the Elasticsearch index;
// fields_only (--fields-only) asks for only the names the provider itself
// understands. kind ("alias" | "field") and field (an alias's provider field)
// are absent on an older api-server.
const LogsListLabelsQuery = `query FetchLogLabels($request: FetchLogLabelRequest!) {
  logs_list_labels(request: $request) {
    label
    kind
    field
  }
}`

var logsListLabelsCmd = &cobra.Command{
	Use:   "list-labels",
	Short: "List log labels",
	Long: `List the log labels (fields) a query can use.

Each label has a kind: "field" is a name the provider itself understands,
so a native query (LogQL, Elasticsearch Query DSL, ...) can use it; "alias"
is a Nudgebee short name, and Field shows the provider field it maps to.
Use --fields-only to list only the names a native query can use.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		accountId, err := resolveAccountID(cmd)
		if err != nil {
			return err
		}

		startTimeStr, _ := cmd.Flags().GetString("start-time")
		endTimeStr, _ := cmd.Flags().GetString("end-time")

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

		index, _ := cmd.Flags().GetString("index")
		fieldsOnly, _ := cmd.Flags().GetBool("fields-only")

		nested := map[string]any{"query": query}
		if index != "" {
			nested["index"] = index
		}
		request := map[string]any{"account_id": accountId, "request": nested}
		if fieldsOnly {
			request["fields_only"] = true
		}
		req := client.NewRequest(LogsListLabelsQuery)
		req.Var("request", request)

		var respData struct {
			LogsListLabels []logLabel `json:"logs_list_labels"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		labels := respData.LogsListLabels
		if fieldsOnly {
			var ok bool
			if labels, ok = onlyProviderFields(labels); !ok {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Warning: this api-server does not report label kinds, so --fields-only cannot tell aliases from provider fields; listing all labels.")
			}
		}

		fields := []format.TableField{{Header: "Label", Field: "Label"}}
		if _, hasKinds := onlyProviderFields(labels); hasKinds {
			fields = append(fields, format.TableField{Header: "Kind", Field: "Kind"}, format.TableField{Header: "Field", Field: "Field"})
		}
		table := format.TabularData{Data: labels, Fields: fields}
		emptyMsg := fmt.Sprintf("No log labels found between %s and %s.", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
		if fieldsOnly {
			emptyMsg = "No provider fields found. The provider may not report which labels are its own fields; run without --fields-only to see all labels."
		}
		return printRows(cmd, table, len(labels), emptyMsg)
	},
}

func init() {
	logsCmd.AddCommand(logsListLabelsCmd)
	logsListLabelsCmd.Flags().String("account-id", "", "Account ID")
	logsListLabelsCmd.Flags().String("start-time", "", "Start time (RFC3339)")
	logsListLabelsCmd.Flags().String("end-time", "", "End time (RFC3339)")
	logsListLabelsCmd.Flags().String("index", "", "Elasticsearch/OpenSearch only: index whose fields to list (other providers ignore it)")
	logsListLabelsCmd.Flags().Bool("fields-only", false, "List only names the provider itself understands (kind \"field\"), i.e. what a native query can use")
}

// logLabel is one logs_list_labels entry. Kind is "alias", "field" or empty:
// empty means unknown (an older api-server, or a provider that does not say),
// never "alias".
type logLabel struct {
	Label string `json:"label"`
	Kind  string `json:"kind,omitempty"`
	Field string `json:"field,omitempty"`
}

// onlyProviderFields keeps the kind "field" entries. The api-server already
// filters when fields_only is set; this guards an api-server that ignores the
// flag. ok is false when no entry has a kind, i.e. the api-server predates it.
func onlyProviderFields(labels []logLabel) (fields []logLabel, ok bool) {
	for _, l := range labels {
		if l.Kind != "" {
			ok = true
		}
		if l.Kind == "field" {
			fields = append(fields, l)
		}
	}
	if !ok {
		return labels, false
	}
	return fields, true
}
