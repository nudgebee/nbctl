package cmd

import (
	"fmt"
	"strings"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

var nubiKbCmd = &cobra.Command{
	Use:   "kb",
	Short: "Manage Knowledge Base vector sources for AI retrieval",
}

type kbItem struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	KBType        string  `json:"kb_type"`
	KBSource      string  `json:"kb_source"`
	Status        string  `json:"status"`
	DocumentCount int     `json:"document_count"`
	DataSizeBytes float64 `json:"data_size_bytes"`
	LastLoadedAt  string  `json:"last_loaded_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type graphqlErrorItem struct {
	Message string `json:"message"`
}

func joinGraphQLErrors(errs []graphqlErrorItem) error {
	if len(errs) == 0 {
		return nil
	}
	var msgs []string
	for _, e := range errs {
		if strings.TrimSpace(e.Message) != "" {
			msgs = append(msgs, strings.TrimSpace(e.Message))
		}
	}
	if len(msgs) == 0 {
		return nil
	}
	return fmt.Errorf("backend error: %s", strings.Join(msgs, "; "))
}

var nubiKbListCmd = &cobra.Command{
	Use:   "list [account-id]",
	Short: "List Knowledge Base sources",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID, err := resolveAccountIDWithPositional(cmd, args)
		if err != nil {
			return err
		}

		req := client.NewRequest(`
			query ListKB($request: ListKBRequest!) {
				ai_list_kb(request: $request) {
					data {
						id
						name
						kb_type
						kb_source
						status
						document_count
						data_size_bytes
						last_loaded_at
						updated_at
					}
					errors {
						message
					}
				}
			}
		`)
		req.Var("request", map[string]any{
			"account_id": accountID,
		})

		var respData struct {
			AiListKb *struct {
				Data   []kbItem           `json:"data"`
				Errors []graphqlErrorItem `json:"errors"`
			} `json:"ai_list_kb"`
		}

		if err := client.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		if respData.AiListKb == nil {
			return fmt.Errorf("empty response from backend")
		}

		if err := joinGraphQLErrors(respData.AiListKb.Errors); err != nil {
			return err
		}

		table := format.TabularData{
			Data: respData.AiListKb.Data,
			Fields: []format.TableField{
				{Header: "KB ID", Field: "ID"},
				{Header: "Name", Field: "Name"},
				{Header: "KB Type", Field: "KBType"},
				{Header: "Status", Field: "Status"},
				{Header: "Documents", Field: "DocumentCount"},
				{Header: "Last Loaded", Field: "LastLoadedAt"},
			},
		}
		format.GetFormat().Print(table)

		return nil
	},
}

var nubiKbGetCmd = &cobra.Command{
	Use:   "get <kb-id>",
	Short: "Get details for a Knowledge Base source",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		kbID := strings.TrimSpace(args[0])
		if kbID == "" {
			return fmt.Errorf("kb-id cannot be empty")
		}

		accountID, err := resolveAccountID(cmd)
		if err != nil {
			return fmt.Errorf("resolving account ID: %w", err)
		}

		req := client.NewRequest(`
			query GetKB($request: GetKBRequest!) {
				ai_get_kb(request: $request) {
					data {
						id
						tenant_id
						account_id
						name
						description
						data_format
						data_filename
						data_size_bytes
						status
						kb_type
						kb_source
						integration_id
						document_count
						last_loaded_at
						created_at
						updated_at
					}
					errors {
						message
					}
				}
			}
		`)
		req.Var("request", map[string]any{
			"account_id": accountID,
			"id":         kbID,
		})

		var respData struct {
			AiGetKb *struct {
				Data *struct {
					ID            string  `json:"id"`
					TenantID      string  `json:"tenant_id"`
					AccountID     string  `json:"account_id"`
					Name          string  `json:"name"`
					Description   string  `json:"description"`
					DataFormat    string  `json:"data_format"`
					DataFilename  string  `json:"data_filename"`
					DataSizeBytes float64 `json:"data_size_bytes"`
					Status        string  `json:"status"`
					KBType        string  `json:"kb_type"`
					KBSource      string  `json:"kb_source"`
					IntegrationID string  `json:"integration_id"`
					DocumentCount int     `json:"document_count"`
					LastLoadedAt  string  `json:"last_loaded_at"`
					CreatedAt     string  `json:"created_at"`
					UpdatedAt     string  `json:"updated_at"`
				} `json:"data"`
				Errors []graphqlErrorItem `json:"errors"`
			} `json:"ai_get_kb"`
		}

		if err := client.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		if respData.AiGetKb == nil {
			return fmt.Errorf("empty response from backend")
		}

		if err := joinGraphQLErrors(respData.AiGetKb.Errors); err != nil {
			return err
		}

		if respData.AiGetKb.Data == nil {
			return fmt.Errorf("knowledge base '%s' not found", kbID)
		}

		format.GetFormat().Print(*respData.AiGetKb.Data)
		return nil
	},
}

type kbActionResponse struct {
	KBID    string `json:"kb_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

var nubiKbSyncCmd = &cobra.Command{
	Use:   "sync <kb-id>",
	Short: "Trigger manual re-indexing / vector embedding sync for a Knowledge Base",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		kbID := strings.TrimSpace(args[0])
		if kbID == "" {
			return fmt.Errorf("kb-id cannot be empty")
		}

		accountID, err := resolveAccountID(cmd)
		if err != nil {
			return fmt.Errorf("resolving account ID: %w", err)
		}

		req := client.NewRequest(`
			mutation SyncKB($request: RetriggerKBRequest!) {
				ai_sync_kb(request: $request) {
					data
					errors {
						message
					}
				}
			}
		`)
		req.Var("request", map[string]any{
			"account_id": accountID,
			"id":         kbID,
		})

		var respData struct {
			AiSyncKb *struct {
				Data   any                `json:"data"`
				Errors []graphqlErrorItem `json:"errors"`
			} `json:"ai_sync_kb"`
		}

		if err := client.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		if respData.AiSyncKb == nil {
			return fmt.Errorf("empty response from backend")
		}

		if err := joinGraphQLErrors(respData.AiSyncKb.Errors); err != nil {
			return err
		}

		format.GetFormat().Print(kbActionResponse{
			KBID:    kbID,
			Status:  "triggered",
			Message: "Knowledge Base vector re-indexing triggered successfully",
		})
		return nil
	},
}

var nubiKbEnableCmd = &cobra.Command{
	Use:   "enable <kb-id>",
	Short: "Enable a Knowledge Base source for AI retrieval",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return toggleKBEnabled(cmd, args[0], true)
	},
}

var nubiKbDisableCmd = &cobra.Command{
	Use:   "disable <kb-id>",
	Short: "Disable a Knowledge Base source from AI retrieval",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return toggleKBEnabled(cmd, args[0], false)
	},
}

func toggleKBEnabled(cmd *cobra.Command, rawKBID string, enabled bool) error {
	kbID := strings.TrimSpace(rawKBID)
	if kbID == "" {
		return fmt.Errorf("kb-id cannot be empty")
	}

	accountID, err := resolveAccountID(cmd)
	if err != nil {
		return fmt.Errorf("resolving account ID: %w", err)
	}

	req := client.NewRequest(`
		mutation UpdateKBEnabled($request: UpdateKBEnabledRequest!) {
			ai_update_kb_enabled(request: $request) {
				data
				errors {
					message
				}
			}
		}
	`)
	req.Var("request", map[string]any{
		"account_id": accountID,
		"kb_id":      kbID,
		"enabled":    enabled,
	})

	var respData struct {
		AiUpdateKbEnabled *struct {
			Data   any                `json:"data"`
			Errors []graphqlErrorItem `json:"errors"`
		} `json:"ai_update_kb_enabled"`
	}

	if err := client.Run(cmd.Context(), req, &respData); err != nil {
		return err
	}

	if respData.AiUpdateKbEnabled == nil {
		return fmt.Errorf("empty response from backend")
	}

	if err := joinGraphQLErrors(respData.AiUpdateKbEnabled.Errors); err != nil {
		return err
	}

	statusStr := "enabled"
	if !enabled {
		statusStr = "disabled"
	}

	format.GetFormat().Print(kbActionResponse{
		KBID:    kbID,
		Status:  statusStr,
		Message: fmt.Sprintf("Knowledge Base successfully %s", statusStr),
	})
	return nil
}

func init() {
	nubiCmd.AddCommand(nubiKbCmd)
	nubiKbCmd.PersistentFlags().String("account-id", "", "Account ID (overrides profile)")

	nubiKbCmd.AddCommand(nubiKbListCmd)
	nubiKbCmd.AddCommand(nubiKbGetCmd)
	nubiKbCmd.AddCommand(nubiKbSyncCmd)
	nubiKbCmd.AddCommand(nubiKbEnableCmd)
	nubiKbCmd.AddCommand(nubiKbDisableCmd)
}
