package cmd

import (
	"fmt"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/spf13/cobra"
)

var securityScanImageCmd = &cobra.Command{
	Use:   "scan-image <image-ref>",
	Short: "Trigger a container image security scan",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		imageRef := args[0]
		accountID, _ := cmd.Flags().GetString("account-id")

		graphqlClient := client.NewClient()

		req := client.NewRequest(`
			mutation SecurityScanImage($request: SecurityScanImageRequest!) {
				security_scan_image(request: $request) {
					status
					message
					scan_id
				}
			}
		`)

		input := map[string]any{
			"image_ref": imageRef,
		}
		if accountID != "" {
			input["account_id"] = accountID
		}

		req.Var("request", input)

		var respData struct {
			SecurityScanImage struct {
				Status  string `json:"status"`
				Message string `json:"message"`
				ScanID  string `json:"scan_id"`
			} `json:"security_scan_image"`
		}

		if err := graphqlClient.Run(cmd.Context(), req, &respData); err != nil {
			return err
		}

		if respData.SecurityScanImage.ScanID != "" {
			fmt.Printf("Scan triggered successfully for %s (Scan ID: %s, Status: %s)\n", imageRef, respData.SecurityScanImage.ScanID, respData.SecurityScanImage.Status)
		} else {
			fmt.Printf("Scan request submitted for %s (Status: %s)\n", imageRef, respData.SecurityScanImage.Status)
		}

		return nil
	},
}

func init() {
	securityCmd.AddCommand(securityScanImageCmd)
	securityScanImageCmd.Flags().String("account-id", "", "Account ID for tenant scoping")
}
