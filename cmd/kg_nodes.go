package cmd

import (
	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var kgNodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "List and search Knowledge Graph service and resource nodes",
	RunE: func(cmd *cobra.Command, args []string) error {
		query, _ := cmd.Flags().GetString("query")
		nodeType, _ := cmd.Flags().GetString("type")
		limit, _ := cmd.Flags().GetInt("limit")

		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query ListKgNodes($query: String, $nodeType: String, $limit: Int) {
				kg_list_nodes(query: $query, node_type: $nodeType, limit: $limit) {
					nodes {
						id
						name
						type
						namespace
						account_id
					}
				}
			}
		`)
		req.Var("query", query)
		if nodeType != "" {
			req.Var("nodeType", nodeType)
		}
		req.Var("limit", limit)

		var respData struct {
			KgListNodes struct {
				Nodes []struct {
					ID        string `json:"id"`
					Name      string `json:"name"`
					Type      string `json:"type"`
					Namespace string `json:"namespace"`
					AccountID string `json:"account_id"`
				} `json:"nodes"`
			} `json:"kg_list_nodes"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.KgListNodes.Nodes,
			Fields: []format.TableField{
				{Header: "Node ID", Field: "ID"},
				{Header: "Name", Field: "Name"},
				{Header: "Type", Field: "Type"},
				{Header: "Namespace", Field: "Namespace"},
				{Header: "Account ID", Field: "AccountID"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	kgCmd.AddCommand(kgNodesCmd)
	kgNodesCmd.Flags().String("query", "", "Filter nodes by name substring")
	kgNodesCmd.Flags().String("type", "", "Filter nodes by type (e.g. service, pod, database)")
	kgNodesCmd.Flags().Int("limit", 50, "Maximum number of nodes to return")
}
