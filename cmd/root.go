package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/nudgebee/nbctl/pkg/config"
	"github.com/nudgebee/nbctl/pkg/format"
	applog "github.com/nudgebee/nbctl/pkg/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	rootCmd = &cobra.Command{
		Use:           "nbctl",
		Short:         "A CLI for interacting with the Nudgebee API",
		Long:          `nbctl is a command-line interface for Nudgebee.`,
		SilenceErrors: true,
		SilenceUsage:  true,
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
	rootCmd.PersistentFlags().StringVarP(&formatVar, "format", "o", "text", "Output format (json; raw for logs query)")
	var outputVar string
	rootCmd.PersistentFlags().StringVar(&outputVar, "output", "", "Output format alias (json)")
	_ = rootCmd.PersistentFlags().MarkHidden("output")
	rootCmd.PersistentFlags().String("profile", "", "Use a specific profile from your config file")
	_ = viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	rootCmd.PersistentFlags().String("http-timeout", "", "Timeout for each API request, e.g. 50s (env NUDGEBEE_HTTP_TIMEOUT; default 30s, 0 disables)")
	_ = viper.BindPFlag("http-timeout", rootCmd.PersistentFlags().Lookup("http-timeout"))

	// Initialize a logger that writes to the command's stderr. Using PersistentPreRunE
	// ensures cmd.ErrOrStderr() is available during execution and in tests.
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		config.InitConfig()
		Logger = log.New(cmd.ErrOrStderr(), "", log.LstdFlags)

		// Set the format from the flag.
		formatValue, _ := cmd.Flags().GetString("format")
		if outputVal, _ := cmd.Flags().GetString("output"); outputVal != "" {
			formatValue = outputVal
		}
		format.GetFormat().Set(formatValue)
		if formatValue == "raw" && cmd.Annotations[rawOutputAnnotation] != "true" {
			return fmt.Errorf("-o raw is only supported by: nbctl logs query")
		}

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

// rawOutputAnnotation marks a command that supports -o raw (the provider's
// own response fragments, unchanged).
const rawOutputAnnotation = "nbctl/raw-output"

// enabledCommandsEnv limits nbctl to a comma-separated list of top-level
// command groups (e.g. "metrics,logs"), for embedding nbctl where only some
// commands are useful. It hides commands; it is not an access control.
const enabledCommandsEnv = "NUDGEBEE_ENABLED_COMMANDS"

// alwaysEnabledCommands stay available whatever enabledCommandsEnv says.
var alwaysEnabledCommands = map[string]bool{"help": true, "version": true, "completion": true}

// restrictCommands removes the top-level commands of root that are not listed
// in enabled (comma-separated). An empty list leaves root unchanged.
func restrictCommands(root *cobra.Command, enabled string) {
	allowed := map[string]bool{}
	for _, name := range strings.Split(enabled, ",") {
		if name = strings.TrimSpace(name); name != "" {
			allowed[name] = true
		}
	}
	if len(allowed) == 0 {
		return
	}
	// Iterate over a copy so the loop does not depend on how cobra stores commands.
	for _, c := range append([]*cobra.Command(nil), root.Commands()...) {
		if !allowed[c.Name()] && !alwaysEnabledCommands[c.Name()] {
			root.RemoveCommand(c)
		}
	}
}

func Execute() {
	restrictCommands(rootCmd, os.Getenv(enabledCommandsEnv))
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
