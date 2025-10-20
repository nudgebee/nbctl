package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/machinebox/graphql"
	"github.com/spf13/cobra"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/format"
)

var (
	listStatus        string
	listCloudProvider string
	listName          string
)

var accountsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := client.NewClient()

		req := graphql.NewRequest(`
			query GetCloudAccounts($where: cloud_accounts_bool_exp) {
				cloud_accounts(where: $where) {
					cloud_provider
					account_type
					id
					account_name
					status
					created_at
					agents {
						last_connected_at
						status
						version
						connection_status
						k8s_provider
						k8s_version
					}
					cloud_account_attrs {
						name
						value
						id
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
			CloudAccounts []struct {
				ID            string `json:"id"`
				AccountName   string `json:"account_name"`
				AccountType   string `json:"account_type"`
				CloudProvider string `json:"cloud_provider"`
				Status        string `json:"status"`
				CreatedAt     string `json:"created_at"`
				Agents        []struct {
					LastConnectedAt  string          `json:"last_connected_at"`
					Status           string          `json:"status"`
					Version          string          `json:"version"`
					ConnectionStatus json.RawMessage `json:"connection_status"`
					K8sProvider      string          `json:"k8s_provider"`
					K8sVersion       string          `json:"k8s_version"`
				} `json:"agents"`
				CloudAccountAttrs []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
					ID    string `json:"id"`
				} `json:"cloud_account_attrs"`
			} `json:"cloud_accounts"`
		}
		if err := client.Run(context.Background(), req, &respData); err != nil {
			return err
		}

		var outputData []struct {
			ID            string
			AccountName   string
			AccountType   string
			CloudProvider string
			Status        string
			Agents        int
			Attributes    int
		}

		for _, acc := range respData.CloudAccounts {
			outputData = append(outputData, struct {
				ID            string
				AccountName   string
				AccountType   string
				CloudProvider string
				Status        string
				Agents        int
				Attributes    int
			}{
				ID:            acc.ID,
				AccountName:   acc.AccountName,
				AccountType:   acc.AccountType,
				CloudProvider: acc.CloudProvider,
				Status:        acc.Status,
				Agents:        len(acc.Agents),
				Attributes:    len(acc.CloudAccountAttrs),
			})
		}

		table := format.TabularData{
			Data: outputData,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "Account Name", Field: "AccountName"},
				{Header: "Account Type", Field: "AccountType"},
				{Header: "Cloud Provider", Field: "CloudProvider"},
				{Header: "Status", Field: "Status"},
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
	accountsListCmd.Flags().StringVar(&listStatus, "status", "", "Filter by status")
	accountsListCmd.Flags().StringVar(&listCloudProvider, "cloud-provider", "", "Filter by cloud provider")
	accountsListCmd.Flags().StringVar(&listName, "name", "", "Filter by name (like match)")
}
