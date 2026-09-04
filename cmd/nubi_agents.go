package cmd

import (
	"strings"

	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

type agentRowDisplay struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Tools       string `json:"tools"`
}

var nubiAgentsCmd = &cobra.Command{
	Use:   "agents [account-id]",
	Short: "List registered AI agents and assigned tools",
	Long:  `Display registered AI agents, descriptions, status, and assigned toolsets.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nubiClient, err := initNubiClientOptionalAccount(args)
		if err != nil {
			return err
		}

		agents, err := nubiClient.ListAgents(cmd.Context())
		if err != nil {
			return err
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(agents)
			return nil
		}

		var rows []agentRowDisplay
		for _, a := range agents {
			rows = append(rows, agentRowDisplay{
				Name:        a.Name,
				Description: a.Description,
				Status:      a.Status,
				Tools:       strings.Join(a.Tools, ", "),
			})
		}

		format.GetFormat().Print(format.TabularData{
			Data: rows,
			Fields: []format.TableField{
				{Header: "NAME", Field: "Name"},
				{Header: "STATUS", Field: "Status"},
				{Header: "DESCRIPTION", Field: "Description"},
				{Header: "TOOLS", Field: "Tools"},
			},
		})
		return nil
	},
}

func init() {
	nubiCmd.AddCommand(nubiAgentsCmd)
}
