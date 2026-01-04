package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var workflowValidateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate a workflow from a YAML file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		yamlContent, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		var workflowData map[string]interface{}
		if err := yaml.Unmarshal(yamlContent, &workflowData); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}

		// Ensure account_id is set if not present in the file
		accountId := viper.GetString("account-id")
		if _, ok := workflowData["account_id"]; !ok {
			workflowData["account_id"] = accountId
		}

		query := `
mutation ValidateWorkflow($request: WorkflowCreateRequest!) {
  workflow_validate(request: $request) {
    message
  }
}
`
		requestVar := map[string]interface{}{
			"account_id": accountId,
			"workflow":   workflowData,
		}

		payload := map[string]interface{}{
			"query": query,
			"variables": map[string]interface{}{
				"request": requestVar,
			},
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal request payload: %w", err)
		}

		httpClient := client.NewHTTPClient()
		endpoint := viper.GetString("endpoint")
		if endpoint == "" {
			endpoint = "https://app.nudgebee.com"
		}
		if endpoint[len(endpoint)-1] == '/' {
			endpoint = endpoint[:len(endpoint)-1]
		}
		graphqlEndpoint := endpoint + "/api/graphql"

		req, err := http.NewRequest(http.MethodPost, graphqlEndpoint, bytes.NewBuffer(payloadBytes))
		if err != nil {
			return fmt.Errorf("failed to create http request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("http request failed: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		var rawResp map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &rawResp); err != nil {
			return fmt.Errorf("failed to parse response JSON: %w", err)
		}

		// Check for top-level errors
		if errorsVal, ok := rawResp["errors"].([]interface{}); ok {
			// Check for specific crash signature in error messages
			for _, errItem := range errorsVal {
				if errMap, ok := errItem.(map[string]interface{}); ok {
					if msg, ok := errMap["message"].(string); ok {
						if strings.Contains(msg, "ActionWebhookErrorResponse") && strings.Contains(msg, "key \"message\" not found") {
							// Attempt to drill down into the actual error from the webhook
							// extensions.internal.response.body.errors[].message
							if extensions, ok := errMap["extensions"].(map[string]interface{}); ok {
								if internal, ok := extensions["internal"].(map[string]interface{}); ok {
									if response, ok := internal["response"].(map[string]interface{}); ok {
										if body, ok := response["body"].(map[string]interface{}); ok {
											if bodyErrs, ok := body["errors"].([]interface{}); ok && len(bodyErrs) > 0 {
												var sb strings.Builder
												sb.WriteString("Workflow is invalid (backend validation failed):\n")
												for _, be := range bodyErrs {
													if beMap, ok := be.(map[string]interface{}); ok {
														if beMsg, ok := beMap["message"].(string); ok {
															sb.WriteString(fmt.Sprintf("- %s\n", beMsg))
														}
													}
												}
												return fmt.Errorf("%s", sb.String())
											}
										}
									}
								}
							}

							// Fallback if drill-down fails
							if viper.GetBool("verbose") {
								jsonBytes, _ := json.MarshalIndent(errorsVal, "", "  ")
								return fmt.Errorf("workflow validation failed (server error):\n%s", string(jsonBytes))
							}
							return fmt.Errorf("workflow validation failed (server error). Run with --verbose to see full details")
						}
					}
				}
			}

			// Convert to JSON to print for generic errors
			jsonBytes, _ := json.MarshalIndent(errorsVal, "", "  ")
			return fmt.Errorf("workflow validation failed:\n%s", string(jsonBytes))
		}

		// Parse data
		dataVal, ok := rawResp["data"].(map[string]interface{})
		if !ok || dataVal == nil {
			// If no errors and no data, something is wrong
			return fmt.Errorf("empty response from server")
		}

		wfValidateVal, ok := dataVal["workflow_validate"].(map[string]interface{})
		if !ok || wfValidateVal == nil {
			return fmt.Errorf("unexpected response structure: workflow_validate missing")
		}

		// Extract fields
		message, _ := wfValidateVal["message"].(string)

		if format.GetFormat().Get() == "json" {
			// Construct response object for JSON output
			respObj := map[string]interface{}{
				"message": message,
				"valid":   true,
			}
			format.GetFormat().Print(respObj)
		} else {
			if message != "" {
				_, _ = fmt.Fprintln(format.GetFormat().GetOutput(), message)
			} else {
				_, _ = fmt.Fprintln(format.GetFormat().GetOutput(), "Workflow is valid")
			}
		}

		return nil
	},
}

func init() {
	workflowCmd.AddCommand(workflowValidateCmd)
}
