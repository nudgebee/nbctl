package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var accountsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		listStatus, _ := cmd.Flags().GetString("status")
		listCloudProvider, _ := cmd.Flags().GetString("cloud-provider")
		listName, _ := cmd.Flags().GetString("name")

		req := client.NewRequest(`
			query GetCloudAccounts($where: CloudAccountWhereRequest!) {
				cloud_accounts: accounts_list(where: $where) {
					rows {
						id
						account_name
						account_type
						cloud_provider
						status
						created_at
						agents
						account_access
						cloud_account_attrs
					}
				}
			}
		`)

		where := make(map[string]any)
		if listStatus != "" {
			where["status"] = map[string]any{"_eq": listStatus}
		}
		if listCloudProvider != "" {
			where["cloud_provider"] = map[string]any{"_eq": listCloudProvider}
		}
		if listName != "" {
			where["account_name"] = map[string]any{"_like": fmt.Sprintf("%%%s%%", listName)}
		}

		req.Var("where", where)

		var respData struct {
			CloudAccounts struct {
				Rows []struct {
					ID                string          `json:"id"`
					AccountName       string          `json:"account_name"`
					AccountType       string          `json:"account_type"`
					CloudProvider     string          `json:"cloud_provider"`
					Status            string          `json:"status"`
					CreatedAt         string          `json:"created_at"`
					Agents            json.RawMessage `json:"agents"`
					AccountAccess     json.RawMessage `json:"account_access"`
					CloudAccountAttrs json.RawMessage `json:"cloud_account_attrs"`
				} `json:"rows"`
			} `json:"cloud_accounts"`
		}
		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		type row struct {
			ID            string
			AccountName   string
			AccountType   string
			CloudProvider string
			Status        string
			CreatedAt     string
			Agents        int
			Attributes    int
		}
		var rows []row
		for _, r := range respData.CloudAccounts.Rows {
			rows = append(rows, row{
				ID:            r.ID,
				AccountName:   r.AccountName,
				AccountType:   r.AccountType,
				CloudProvider: r.CloudProvider,
				Status:        r.Status,
				CreatedAt:     r.CreatedAt,
				Agents:        countJSONArray(r.Agents),
				Attributes:    countJSONArray(r.CloudAccountAttrs),
			})
		}

		table := format.TabularData{
			Data: rows,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "Account Name", Field: "AccountName"},
				{Header: "Account Type", Field: "AccountType"},
				{Header: "Cloud Provider", Field: "CloudProvider"},
				{Header: "Status", Field: "Status"},
				{Header: "Created At", Field: "CreatedAt"},
				{Header: "Agents", Field: "Agents"},
				{Header: "Attributes", Field: "Attributes"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	accountsCmd.AddCommand(accountsListCmd)
	accountsListCmd.Flags().String("status", "", "Filter by status")
	accountsListCmd.Flags().String("cloud-provider", "", "Filter by cloud provider")
	accountsListCmd.Flags().String("name", "", "Filter by name (like match)")
}

func countJSONArray(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	payload := []byte(raw)
	// jsonb fields are sometimes returned as a JSON-encoded string; peel one layer.
	if len(payload) > 0 && payload[0] == '"' {
		var s string
		if err := json.Unmarshal(payload, &s); err == nil {
			payload = []byte(s)
		}
	}
	var arr []any
	if err := json.Unmarshal(payload, &arr); err != nil {
		return 0
	}
	return len(arr)
}
