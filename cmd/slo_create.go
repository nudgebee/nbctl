package cmd

import (
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/spf13/cobra"
)

var sloCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create an SLO configuration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		target, _ := cmd.Flags().GetFloat64("target")
		window, _ := cmd.Flags().GetString("window")
		service, _ := cmd.Flags().GetString("service")

		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			mutation CreateSloConfig($request: SloConfigCreateInput!) {
				slo_config_create(request: $request) {
					id
					name
					status
				}
			}
		`)

		input := map[string]any{
			"name":         name,
			"target":       target,
			"time_window":  window,
			"service_name": service,
		}
		req.Var("request", input)

		var respData struct {
			SloConfigCreate struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Status string `json:"status"`
			} `json:"slo_config_create"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		fmt.Printf("SLO '%s' created successfully (ID: %s)\n", name, respData.SloConfigCreate.ID)
		return nil
	},
}

func init() {
	sloCmd.AddCommand(sloCreateCmd)
	sloCreateCmd.Flags().Float64("target", 99.9, "SLO target percentage (e.g. 99.9)")
	sloCreateCmd.Flags().String("window", "30d", "Time window (e.g. 7d, 30d)")
	sloCreateCmd.Flags().String("service", "", "Target service name (required)")
	_ = sloCreateCmd.MarkFlagRequired("service")
}
