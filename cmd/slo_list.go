package cmd

import (
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	sloListWorkloadName      string
	sloListWorkloadNamespace string
	sloListEnabled           string
)

var sloListCmd = &cobra.Command{
	Use:   "list",
	Short: "List SLO configurations",
	Long:  `List Service Level Objective configurations for the current account.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID := viper.GetString("account-id")
		if accountID == "" {
			return fmt.Errorf("account-id is required; set it in your profile with `nbctl configure add`")
		}

		where := map[string]any{
			"cloud_account_id": map[string]any{"_eq": accountID},
		}
		if sloListWorkloadName != "" {
			where["workload_name"] = map[string]any{"_eq": sloListWorkloadName}
		}
		if sloListWorkloadNamespace != "" {
			where["workload_namespace"] = map[string]any{"_eq": sloListWorkloadNamespace}
		}
		if sloListEnabled != "" {
			if sloListEnabled != "true" && sloListEnabled != "false" {
				return fmt.Errorf("invalid value for --enabled: %q (must be true or false)", sloListEnabled)
			}
			where["enabled"] = map[string]any{"_eq": sloListEnabled == "true"}
		}

		query := `
query ListSLOs($where: SLOConfigWhereRequest!) {
  slo_config_v2(where: $where, order_by: [{column: "updated_at", order: desc}]) {
    rows {
      id
      name
      workload_name
      workload_namespace
      cloud_account_id
      goal
      threshold
      window
      enabled
      created_at
      updated_at
    }
  }
}
`
		req := client.NewRequest(query)
		req.Var("where", where)

		var respData struct {
			SLOConfigV2 struct {
				Rows []struct {
					ID                string `json:"id"`
					Name              string `json:"name"`
					WorkloadName      string `json:"workload_name"`
					WorkloadNamespace string `json:"workload_namespace"`
					CloudAccountID    string `json:"cloud_account_id"`
					Goal              string `json:"goal"`
					Threshold         string `json:"threshold"`
					Window            string `json:"window"`
					Enabled           bool   `json:"enabled"`
					CreatedAt         string `json:"created_at"`
					UpdatedAt         string `json:"updated_at"`
				} `json:"rows"`
			} `json:"slo_config_v2"`
		}

		c := client.NewClient()
		if err := c.Run(cmd.Context(), req, &respData); err != nil {
			return fmt.Errorf("failed to list SLOs: %w", err)
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(respData.SLOConfigV2.Rows)
			return nil
		}

		format.GetFormat().Print(format.TabularData{
			Data: respData.SLOConfigV2.Rows,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "Name", Field: "Name"},
				{Header: "Namespace", Field: "WorkloadNamespace"},
				{Header: "Workload", Field: "WorkloadName"},
				{Header: "Goal", Field: "Goal"},
				{Header: "Threshold", Field: "Threshold"},
				{Header: "Window", Field: "Window"},
				{Header: "Enabled", Field: "Enabled"},
				{Header: "Updated At", Field: "UpdatedAt"},
			},
		})

		return nil
	},
}

func init() {
	sloCmd.AddCommand(sloListCmd)
	sloListCmd.Flags().StringVar(&sloListWorkloadName, "workload", "", "Filter by workload name")
	sloListCmd.Flags().StringVar(&sloListWorkloadNamespace, "namespace", "", "Filter by workload namespace")
	sloListCmd.Flags().StringVar(&sloListEnabled, "enabled", "", "Filter by enabled status (true|false)")
}
