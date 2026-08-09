package cmd

import (
	"strings"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var workflowTaskDefinitionsCmd = &cobra.Command{
	Use:   "task-definitions",
	Short: "List supported workflow task definitions and action schemas",
	RunE: func(cmd *cobra.Command, args []string) error {
		nameFilter, _ := cmd.Flags().GetString("name")
		limit, _ := cmd.Flags().GetInt("limit")

		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query ListWorkflowTaskDefinitions($params: WorkflowTaskDefinitionListRequest!) {
				workflow_list_taskdefinitions(params: $params) {
					tasks {
						name
						description
						aliases
					}
				}
			}
		`)

		params := map[string]any{
			"limit": limit,
		}
		if nameFilter != "" {
			params["name"] = nameFilter
		}
		req.Var("params", params)

		var respData struct {
			WorkflowListTaskdefinitions struct {
				Tasks []struct {
					Name        string   `json:"name"`
					Description string   `json:"description"`
					Aliases     []string `json:"aliases"`
				} `json:"tasks"`
			} `json:"workflow_list_taskdefinitions"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		type taskRow struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Aliases     string `json:"aliases"`
		}
		rows := make([]taskRow, 0, len(respData.WorkflowListTaskdefinitions.Tasks))
		for _, t := range respData.WorkflowListTaskdefinitions.Tasks {
			aliasesStr := "-"
			if len(t.Aliases) > 0 {
				aliasesStr = strings.Join(t.Aliases, ", ")
			}
			rows = append(rows, taskRow{
				Name:        t.Name,
				Description: t.Description,
				Aliases:     aliasesStr,
			})
		}

		table := format.TabularData{
			Data: rows,
			Fields: []format.TableField{
				{Header: "Task Name", Field: "Name"},
				{Header: "Description", Field: "Description"},
				{Header: "Aliases", Field: "Aliases"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	workflowCmd.AddCommand(workflowTaskDefinitionsCmd)
	workflowTaskDefinitionsCmd.Flags().String("name", "", "Filter task definitions by name")
	workflowTaskDefinitionsCmd.Flags().Int("limit", 100, "Maximum number of task definitions to return")
}
