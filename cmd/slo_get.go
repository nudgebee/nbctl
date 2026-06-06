package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var sloGetReport bool

var sloGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get SLO configuration details",
	Long: `Fetch a single SLO configuration by ID. Use --report to also fetch the latest
compliance report (burn rate, event budgets, status).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		accountID, _ := cmd.Flags().GetString("account-id")
		if accountID == "" {
			accountID = viper.GetString("account-id")
		}
		if accountID == "" {
			return fmt.Errorf("account-id is required; set it in your profile with `nbctl configure add` or pass --account-id")
		}

		configQuery := `
query GetSLOConfig($where: SLOConfigWhereRequest!) {
  slo_config_v2(where: $where, limit: 1) {
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
		req := client.NewRequest(configQuery)
		req.Var("where", map[string]any{
			"id":               map[string]any{"_eq": id},
			"cloud_account_id": map[string]any{"_eq": accountID},
		})

		var configResp struct {
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
		if err := c.Run(cmd.Context(), req, &configResp); err != nil {
			return fmt.Errorf("failed to get SLO: %w", err)
		}

		if len(configResp.SLOConfigV2.Rows) == 0 {
			return fmt.Errorf("SLO %s not found", id)
		}
		cfg := configResp.SLOConfigV2.Rows[0]

		var reportRow *sloReportRow
		if sloGetReport {
			var reportErr error
			reportRow, reportErr = fetchLatestSLOReport(cmd.Context(), c, accountID, id)
			if reportErr != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not fetch SLO report: %v\n", reportErr)
			}
		}

		if format.GetFormat().Get() == "json" {
			out := map[string]any{"config": cfg}
			if reportRow != nil {
				out["report"] = reportRow
			}
			format.GetFormat().Print(out)
			return nil
		}

		w := format.GetFormat().GetOutput()
		_, _ = fmt.Fprintf(w, "ID:        %s\n", cfg.ID)
		_, _ = fmt.Fprintf(w, "Name:      %s\n", cfg.Name)
		_, _ = fmt.Fprintf(w, "Namespace: %s\n", cfg.WorkloadNamespace)
		_, _ = fmt.Fprintf(w, "Workload:  %s\n", cfg.WorkloadName)
		_, _ = fmt.Fprintf(w, "Goal:      %s\n", cfg.Goal)
		_, _ = fmt.Fprintf(w, "Threshold: %s\n", cfg.Threshold)
		_, _ = fmt.Fprintf(w, "Window:    %s\n", cfg.Window)
		_, _ = fmt.Fprintf(w, "Enabled:   %v\n", cfg.Enabled)
		_, _ = fmt.Fprintf(w, "Created:   %s\n", cfg.CreatedAt)
		_, _ = fmt.Fprintf(w, "Updated:   %s\n", cfg.UpdatedAt)

		if reportRow != nil {
			_, _ = fmt.Fprintf(w, "\nLatest Report:\n")
			_, _ = fmt.Fprintf(w, "  Status:               %s\n", reportRow.Status)
			_, _ = fmt.Fprintf(w, "  Error Budget Burn:    %v\n", reportRow.ErrorBudgetBurnRate)
			_, _ = fmt.Fprintf(w, "  Events (good/bad/all): %d / %d / %d\n", reportRow.GoodEventsCount, reportRow.BadEventsCount, reportRow.EventsCount)
			_, _ = fmt.Fprintf(w, "  Updated At:           %s\n", reportRow.UpdatedAt)
		}

		return nil
	},
}

type sloReportRow struct {
	ID                  string          `json:"id"`
	ConfigID            string          `json:"config_id"`
	CloudAccountID      string          `json:"cloud_account_id"`
	WorkloadName        string          `json:"workload_name"`
	WorkloadNamespace   string          `json:"workload_namespace"`
	Status              string          `json:"status"`
	ErrorBudgetBurnRate float64         `json:"error_budget_burn_rate"`
	EventsCount         int             `json:"events_count"`
	GoodEventsCount     int             `json:"good_events_count"`
	BadEventsCount      int             `json:"bad_events_count"`
	SLOConfig           json.RawMessage `json:"slo_config"`
	CreatedAt           string          `json:"created_at"`
	UpdatedAt           string          `json:"updated_at"`
}

func fetchLatestSLOReport(ctx context.Context, c *client.Client, accountID, configID string) (*sloReportRow, error) {
	query := `
query SLOReport($where: SLOReportWhereRequest!) {
  slo_report_v2(where: $where, order_by: [{column: "updated_at", order: desc}], limit: 1) {
    rows {
      id
      config_id
      cloud_account_id
      workload_name
      workload_namespace
      status
      error_budget_burn_rate
      events_count
      good_events_count
      bad_events_count
      slo_config
      created_at
      updated_at
    }
  }
}
`
	req := client.NewRequest(query)
	req.Var("where", map[string]any{
		"cloud_account_id": map[string]any{"_eq": accountID},
		"config_id":        map[string]any{"_eq": configID},
	})

	var resp struct {
		SLOReportV2 struct {
			Rows []sloReportRow `json:"rows"`
		} `json:"slo_report_v2"`
	}
	if err := c.Run(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.SLOReportV2.Rows) == 0 {
		return nil, nil
	}
	return &resp.SLOReportV2.Rows[0], nil
}

func init() {
	sloCmd.AddCommand(sloGetCmd)
	sloGetCmd.Flags().String("account-id", "", "Account ID (overrides profile)")
	sloGetCmd.Flags().BoolVar(&sloGetReport, "report", false, "Also fetch the latest compliance report")
}
