package cmd

import (
	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var workflowMcpToolsCmd = &cobra.Command{
	Use:   "mcp-tools",
	Short: "List MCP tools available for workflow task steps",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query ListWorkflowMcpTools {
				workflow_list_mcp_tools {
					tools {
						name
						description
						category
					}
				}
			}
		`)

		var respData struct {
			WorkflowListMcpTools struct {
				Tools []struct {
					Name        string `json:"name"`
					Description string `json:"description"`
					Category    string `json:"category"`
				} `json:"tools"`
			} `json:"workflow_list_mcp_tools"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.WorkflowListMcpTools.Tools,
			Fields: []format.TableField{
				{Header: "Tool Name", Field: "Name"},
				{Header: "Category", Field: "Category"},
				{Header: "Description", Field: "Description"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	workflowCmd.AddCommand(workflowMcpToolsCmd)
}
