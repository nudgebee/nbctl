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

type functionRowDisplay struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Variables   string `json:"variables"`
}

var nubiFunctionsCmd = &cobra.Command{
	Use:   "functions [account-id]",
	Short: "List prompt template functions and variables",
	Long:  `Display prompt template functions, descriptions, status, and required variables.`,
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

		functions, err := nubiClient.ListFunctions()
		if err != nil {
			return fmt.Errorf("failed to list functions: %w", err)
		}

		if format.GetFormat().Get() == "json" {
			format.GetFormat().Print(functions)
			return nil
		}

		var rows []functionRowDisplay
		for _, f := range functions {
			rows = append(rows, functionRowDisplay{
				Name:        f.Name,
				Description: f.Description,
				Status:      f.Status,
				Variables:   strings.Join(f.Variables, ", "),
			})
		}

		format.GetFormat().Print(format.TabularData{
			Data: rows,
			Fields: []format.TableField{
				{Header: "NAME", Field: "Name"},
				{Header: "STATUS", Field: "Status"},
				{Header: "DESCRIPTION", Field: "Description"},
				{Header: "VARIABLES", Field: "Variables"},
			},
		})
		return nil
	},
}

func init() {
	nubiCmd.AddCommand(nubiFunctionsCmd)
}
