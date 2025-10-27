package cmd

import (
	"context"
	"fmt"

	"github.com/machinebox/graphql"
	"github.com/spf13/cobra"
	"nudgebee.com/nbctl/pkg/client"
	"nudgebee.com/nbctl/pkg/format"
)

var (
	createAccountName   string
	createCloudProvider string
	createAccountType   string
	createAccountEnv    string
)

var accountsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Nudgebee account",
	Long:  `Create a new Nudgebee account with specified details.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		graphqlClient := client.NewClient()

		accountName := createAccountName
		cloudProvider := createCloudProvider
		accountType := createAccountType
		accountEnv := createAccountEnv

		if accountEnv == "" {
			accountEnv = "dev"
		} else if accountEnv != "dev" && accountEnv != "prod" {
			return fmt.Errorf("invalid value for --env: %s. Must be 'dev' or 'prod'", accountEnv)
		}

		if accountType == "kubernetes" && cloudProvider == "" {
			cloudProvider = "K8s"
		} else if accountType == "kubernetes" && cloudProvider != "K8s" {
			return fmt.Errorf("invalid value for --cloud-provider: %s. Must be 'K8s' when --account-type is 'kubernetes'", cloudProvider)
		}

		if cloudProvider == "" {
			return fmt.Errorf("--cloud-provider is required unless --account-type is 'kubernetes'")
		}

		object := map[string]any{
			"account_name":   accountName,
			"cloud_provider": cloudProvider,
			"account_type":   accountType,
			"account_env":    accountEnv,
		}

		req := graphql.NewRequest(`
			mutation CreateAccount($object: cloud_accounts_insert_one_input!) {
				cloud_accounts_insert_one(object: $object) {
					id
					access_key
					access_secret
				}
			}
		`)

		req.Var("object", object)

		var respData struct {
			CloudAccountsInsertOne struct {
				ID           string `json:"id"`
				AccessKey    string `json:"access_key"`
				AccessSecret string `json:"access_secret"`
			} `json:"cloud_accounts_insert_one"`
		}

		if err := graphqlClient.Run(context.Background(), req, &respData); err != nil {
			if gqlErrs, ok := err.(client.GraphQLErrors); ok {
				for _, e := range gqlErrs {
					if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "GraphQL Error: %s\n", e.Message); err != nil {
						return err
					}
				}
				return fmt.Errorf("failed to create account due to GraphQL errors")
			} else {
				return fmt.Errorf("failed to create account: %w", err)
			}
		}

		if respData.CloudAccountsInsertOne.ID == "" {
			return fmt.Errorf("account creation failed, no ID returned")
		}
		output := struct {
			ID           string `json:"id"`
			AccessKey    string `json:"access_key"`
			AccessSecret string `json:"access_secret"`
		}{
			ID:           respData.CloudAccountsInsertOne.ID,
			AccessKey:    respData.CloudAccountsInsertOne.AccessKey,
			AccessSecret: respData.CloudAccountsInsertOne.AccessSecret,
		}

		format.GetFormat().Print(output)

		return nil
	},
}

func init() {

	accountsCreateCmd.Flags().StringVar(&createAccountName, "name", "", "Name of the account")
	accountsCreateCmd.Flags().StringVar(&createCloudProvider, "cloud-provider", "", "Cloud provider (e.g., K8s)")
	accountsCreateCmd.Flags().StringVar(&createAccountType, "account-type", "", "Type of the account (e.g., kubernetes)")
	accountsCreateCmd.Flags().StringVar(&createAccountEnv, "env", "", "Environment of the account (e.g., dev, prod)")

	_ = accountsCreateCmd.MarkFlagRequired("name")
	_ = accountsCreateCmd.MarkFlagRequired("account-type")
}
