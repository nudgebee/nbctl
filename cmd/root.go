package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"nudgebee.com/nbctl/pkg/config"
	"nudgebee.com/nbctl/pkg/format"
	applog "nudgebee.com/nbctl/pkg/log"
)

var (
	rootCmd = &cobra.Command{
		Use:   "nbctl",
		Short: "A CLI for interacting with the Nudgebee API",
		Long:  `nbctl is a command-line interface for Nudgebee.`,
	}

	completionCmd = &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Long: `To load completions:

Bash:

  $ source <(nbctl completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ nbctl completion bash > /etc/bash_completion.d/nbctl
  # macOS:
  $ nbctl completion bash > /usr/local/etc/bash_completion.d/nbctl

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ nbctl completion zsh > "${fpath[1]}/_nbctl"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ nbctl completion fish | source

  # To load completions for each session, execute once:
  $ nbctl completion fish > ~/.config/fish/completions/nbctl.fish

Powershell:

  PS> nbctl completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> nbctl completion powershell > nbctl.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				if err := cmd.Root().GenBashCompletion(os.Stdout); err != nil {
					fmt.Fprintf(os.Stderr, "Error generating bash completion: %v\n", err)
				}
			case "zsh":
				if err := cmd.Root().GenZshCompletion(os.Stdout); err != nil {
					fmt.Fprintf(os.Stderr, "Error generating zsh completion: %v\n", err)
				}
			case "fish":
				if err := cmd.Root().GenFishCompletion(os.Stdout, true); err != nil {
					fmt.Fprintf(os.Stderr, "Error generating fish completion: %v\n", err)
				}
			case "powershell":
				if err := cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout); err != nil {
					fmt.Fprintf(os.Stderr, "Error generating powershell completion: %v\n", err)
				}
			}
		},
	}

	// Logger is the package-level logger initialized in PersistentPreRunE.
	// It writes to the command's stderr stream so diagnostics don't pollute stdout.
	Logger   *log.Logger
	logLevel string
)

var Version string = "dev"

func init() {
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(versionCmd)
	// Add a persistent flag for log level. We don't wire levels into std log here,
	// but keep the flag for future use or to be read by other code.
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug|info|warn|error)")

	// Add a persistent flag for verbose logging.
	rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose logging for GraphQL requests and responses")
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// Add a persistent flag for the output format.
	var formatVar string
	rootCmd.PersistentFlags().StringVar(&formatVar, "format", "text", "Output format (json)")
	rootCmd.PersistentFlags().String("profile", "", "Use a specific profile from your config file")
	_ = viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))

	// Initialize a logger that writes to the command's stderr. Using PersistentPreRunE
	// ensures cmd.ErrOrStderr() is available during execution and in tests.
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		config.InitConfig()
		Logger = log.New(cmd.ErrOrStderr(), "", log.LstdFlags)

		// Set the format from the flag.
		formatValue, _ := cmd.Flags().GetString("format")
		format.GetFormat().Set(formatValue)

		// Initialize the logger.
		applog.InitLogger()

		// Check if the command or its parent is one that doesn't require configuration.
		if cmd.Name() != "help" && cmd.Name() != "version" && cmd.Parent().Name() != "configure" && cmd.Name() != "configure" {
			if !config.IsConfigured() {
				return fmt.Errorf("nbctl is not configured. Please run 'nbctl configure' to set up your credentials")
			}
		}

		return nil
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}