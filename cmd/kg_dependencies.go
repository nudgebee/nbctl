package cmd

import (
	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var kgDependenciesCmd = &cobra.Command{
	Use:   "dependencies",
	Short: "Inspect and manage manual dependency overrides",
}

var kgDependenciesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List manual dependency override rules",
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query ListManualDependencies {
				kg_list_manual_dependencies {
					rows {
						id
						source_service
						target_service
						relation
						status
						created_by
						created_at
					}
				}
			}
		`)

		var respData struct {
			KgListManualDependencies struct {
				Rows []struct {
					ID            string `json:"id"`
					SourceService string `json:"source_service"`
					TargetService string `json:"target_service"`
					Relation      string `json:"relation"`
					Status        string `json:"status"`
					CreatedBy     string `json:"created_by"`
					CreatedAt     string `json:"created_at"`
				} `json:"rows"`
			} `json:"kg_list_manual_dependencies"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.KgListManualDependencies.Rows,
			Fields: []format.TableField{
				{Header: "ID", Field: "ID"},
				{Header: "Source Service", Field: "SourceService"},
				{Header: "Target Service", Field: "TargetService"},
				{Header: "Relation", Field: "Relation"},
				{Header: "Status", Field: "Status"},
				{Header: "Created By", Field: "CreatedBy"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

var kgDependenciesResolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Add or resolve a manual dependency override",
	RunE: func(cmd *cobra.Command, args []string) error {
		source, _ := cmd.Flags().GetString("source")
		target, _ := cmd.Flags().GetString("target")
		relation, _ := cmd.Flags().GetString("relation")

		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			mutation ResolveManualDependency($source: String!, $target: String!, $relation: String) {
				kg_resolve_manual_dependency(source_service: $source, target_service: $target, relation: $relation) {
					id
					status
					message
				}
			}
		`)
		req.Var("source", source)
		req.Var("target", target)
		req.Var("relation", relation)

		var respData struct {
			KgResolveManualDependency struct {
				ID      string `json:"id"`
				Status  string `json:"status"`
				Message string `json:"message"`
			} `json:"kg_resolve_manual_dependency"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		format.GetFormat().Print(respData.KgResolveManualDependency)
		return nil
	},
}

func init() {
	kgCmd.AddCommand(kgDependenciesCmd)
	kgDependenciesCmd.AddCommand(kgDependenciesListCmd)
	kgDependenciesCmd.AddCommand(kgDependenciesResolveCmd)

	kgDependenciesResolveCmd.Flags().String("source", "", "Source service name (required)")
	kgDependenciesResolveCmd.Flags().String("target", "", "Target service name (required)")
	kgDependenciesResolveCmd.Flags().String("relation", "calls", "Dependency relationship type (default: calls)")
	_ = kgDependenciesResolveCmd.MarkFlagRequired("source")
	_ = kgDependenciesResolveCmd.MarkFlagRequired("target")
}
