package cmd

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/nudgebee/nbctl/pkg/nubi"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		var accountID string
		if len(args) > 0 {
			accountID = args[0]
		} else {
			accountID = viper.GetString("account-id")
		}

		if accountID == "" {
			return fmt.Errorf("account-id is required, please provide it as an argument or set it in your config file")
		}

		username := viper.GetString("username")
		if username == "" {
			return fmt.Errorf("username is required, please set it in your config file")
		}

		endpoint := viper.GetString("endpoint")
		sessionID := uuid.New().String()
		nubiClient := nubi.New(client.NewClient(), accountID, username, sessionID, endpoint)

		agents, err := nubiClient.ListAgents()
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
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
