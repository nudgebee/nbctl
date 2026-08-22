package cmd

import (
	"strings"

	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
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
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nubiClient, err := initNubiClient(args)
		if err != nil {
			return err
		}

		functions, err := nubiClient.ListFunctions()
		if err != nil {
			return err
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
