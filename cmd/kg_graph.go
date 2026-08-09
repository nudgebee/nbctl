package cmd

import (
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var kgGraphCmd = &cobra.Command{
	Use:   "graph <node-id>",
	Short: "Traverse dependency graph starting from a target node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeID := args[0]
		depth, _ := cmd.Flags().GetInt("depth")

		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			query TraverseKgGraph($nodeId: String!, $depth: Int) {
				kg_graph(node_id: $nodeId, depth: $depth) {
					source_id
					target_id
					relation
					source_type
				}
			}
		`)
		req.Var("nodeId", nodeID)
		req.Var("depth", depth)

		var respData struct {
			KgGraph []struct {
				SourceID   string `json:"source_id"`
				TargetID   string `json:"target_id"`
				Relation   string `json:"relation"`
				SourceType string `json:"source_type"`
			} `json:"kg_graph"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		if len(respData.KgGraph) == 0 {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "No graph relationships found for node '"+nodeID+"'")
			return nil
		}

		table := format.TabularData{
			Data: respData.KgGraph,
			Fields: []format.TableField{
				{Header: "Source Node", Field: "SourceID"},
				{Header: "Relation", Field: "Relation"},
				{Header: "Target Node", Field: "TargetID"},
				{Header: "Source Type", Field: "SourceType"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

func init() {
	kgCmd.AddCommand(kgGraphCmd)
	kgGraphCmd.Flags().Int("depth", 2, "Traversal depth limit")
}
