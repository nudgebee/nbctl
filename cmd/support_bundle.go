package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/nudgebee/nbctl/pkg/client"
	"github.com/nudgebee/nbctl/pkg/config"
	"github.com/nudgebee/nbctl/pkg/diagnostics"
	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Offline collection limits; they match the server's.
const (
	offlineSectionTimeout = 2 * time.Minute
	offlineMaxEvents      = 300
)

var offlineLogLimits = diagnostics.LogLimits{
	Window: 6 * time.Hour, MaxBytes: 2 << 20, TailLines: 50, MaxProblems: 200, Parallel: 6,
}

// DefaultSupportPublicKey is the age recipient an offline bundle is encrypted
// to when --recipient is not given. Empty until the support key is published;
// the bundle is then written unencrypted, with a warning.
const DefaultSupportPublicKey = ""

var onlineSections = []string{"pods", "logs", "broker", "agents", "database"}

type supportBundleResult struct {
	Path      string `json:"path"`
	Encrypted bool   `json:"encrypted"`
	Mode      string `json:"mode"`
}

var supportBundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Write a diagnostics bundle to send to Nudgebee support",
	Long: `Write a diagnostics bundle of a self-hosted install: pod status, error and
warning log lines, broker queues, agent connections and migration state.
Secrets are masked. The file is encrypted to the support key the server is
configured with, or to --recipient; with neither it is written unencrypted,
with a warning.

By default the Nudgebee server builds the bundle (same as Admin > Diagnostics).
With --offline, nbctl collects pods, events and logs itself through kubectl and
your kubeconfig. Use that when the server itself is down.`,
	// --offline must work on a machine where nbctl was never configured.
	Annotations: map[string]string{configOptionalAnnotation: "true"},
	Example: `  nbctl support bundle --case-id 1234
  nbctl support bundle --offline --namespace nudgebee --context my-cluster`,
	RunE: func(cmd *cobra.Command, args []string) error {
		offline, _ := cmd.Flags().GetBool("offline")
		caseID, _ := cmd.Flags().GetString("case-id")
		outDir, _ := cmd.Flags().GetString("output-dir")
		recipient, _ := cmd.Flags().GetString("recipient")
		// Fail before collecting, not after.
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return fmt.Errorf("output directory: %w", err)
		}

		var (
			manifest any
			sections map[string]json.RawMessage
			key      string
			err      error
		)
		if offline {
			namespace, _ := cmd.Flags().GetString("namespace")
			kubeContext, _ := cmd.Flags().GetString("context")
			manifest, sections, err = collectOffline(cmd.Context(), diagnostics.NewKubectl(kubeContext, namespace), caseID, time.Now())
			key = DefaultSupportPublicKey
		} else {
			if !config.IsConfigured() {
				return fmt.Errorf("nbctl is not configured. Run 'nbctl configure', or use --offline to collect with kubectl")
			}
			requested, _ := cmd.Flags().GetStringSlice("sections")
			for _, s := range requested {
				if !slices.Contains(onlineSections, s) {
					return fmt.Errorf("unknown section %q (valid: %v)", s, onlineSections)
				}
			}
			manifest, sections, key, err = collectOnline(requested, caseID)
		}
		if err != nil {
			return err
		}
		if recipient != "" {
			key = recipient
		}

		files, err := diagnostics.Files(manifest, sections)
		if err != nil {
			return err
		}
		path, encrypted, err := diagnostics.Write(outDir, files, key, time.Now())
		if err != nil {
			return err
		}
		if !encrypted {
			fmt.Fprintln(os.Stderr, "Warning: no support key is configured, so the bundle is NOT encrypted. Send it only over a channel you trust, or pass --recipient.")
		}
		fmt.Fprintln(os.Stderr, "Review the bundle before you send it: secrets are masked, but masking cannot recognise every kind of customer data in a log line.")
		mode := "server"
		if offline {
			mode = "offline"
		}
		format.GetFormat().Print(supportBundleResult{Path: path, Encrypted: encrypted, Mode: mode})
		return nil
	},
}

// collectOnline asks the server to build the bundle (diagnostics_generate_bundle).
func collectOnline(sections []string, caseID string) (json.RawMessage, map[string]json.RawMessage, string, error) {
	// Generation reads every pod's logs; give it longer than the default
	// request timeout unless the user set one.
	if viper.GetString("http-timeout") == "" {
		viper.Set("http-timeout", "200s")
	}
	req := client.NewRequest(`
		mutation DiagnosticsGenerateBundle($sections: [String!], $caseId: String) {
			diagnostics_generate_bundle(sections: $sections, case_id: $caseId) {
				manifest
				sections
				support_public_key
			}
		}`)
	req.Var("sections", sections)
	req.Var("caseId", caseID)

	var resp struct {
		Bundle struct {
			Manifest         json.RawMessage            `json:"manifest"`
			Sections         map[string]json.RawMessage `json:"sections"`
			SupportPublicKey string                     `json:"support_public_key"`
		} `json:"diagnostics_generate_bundle"`
	}
	if err := client.NewClient().Run(context.Background(), req, &resp); err != nil {
		return nil, nil, "", fmt.Errorf("the server could not build the bundle (if it is down, use --offline): %w", err)
	}
	if len(resp.Bundle.Manifest) == 0 {
		return nil, nil, "", fmt.Errorf("the server returned no bundle; it may be older than this command (try --offline)")
	}
	return resp.Bundle.Manifest, resp.Bundle.Sections, resp.Bundle.SupportPublicKey, nil
}

func runSection(parent context.Context, fn func(ctx context.Context) (any, error)) diagnostics.SectionResult {
	ctx, cancel := context.WithTimeout(parent, offlineSectionTimeout)
	defer cancel()
	start := time.Now()
	data, err := fn(ctx)
	res := diagnostics.SectionResult{Status: "ok", Data: data, DurationMs: time.Since(start).Milliseconds()}
	if err != nil {
		res.Status, res.Error, res.Data = "error", diagnostics.Redact(err.Error()), nil
	}
	return res
}

// collectOffline builds the pods and logs sections through kubectl. The
// broker, agent and database sections need the server and are left out.
func collectOffline(ctx context.Context, k *diagnostics.Kubectl, caseID string, now time.Time) (diagnostics.Manifest, map[string]json.RawMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	manifest := diagnostics.Manifest{
		Format: diagnostics.FormatName, GeneratedAt: now.UTC(), GeneratedBy: "nbctl --offline", CaseID: caseID,
		Sections: []string{"pods", "logs"}, LogWindow: offlineLogLimits.Window.String(),
		Note: "Collected by nbctl from the Kubernetes API (server unavailable). Logs hold only error/warning lines and a short tail, with secrets masked; review before sharing.",
	}

	var pods []diagnostics.PodStatus
	podsResult := runSection(ctx, func(ctx context.Context) (any, error) {
		s, p, err := k.CollectPods(ctx, offlineMaxEvents)
		pods = p
		return s, err
	})
	logsResult := diagnostics.SectionResult{Status: "error", Error: "pod list unavailable: " + podsResult.Error}
	if podsResult.Status == "ok" {
		logsResult = runSection(ctx, func(ctx context.Context) (any, error) {
			return k.CollectLogs(ctx, pods, offlineLogLimits), nil
		})
	}
	if podsResult.Status != "ok" && logsResult.Status != "ok" {
		return manifest, nil, fmt.Errorf("could not read namespace %q: %s", k.Namespace, podsResult.Error)
	}

	sections := map[string]json.RawMessage{}
	for name, r := range map[string]diagnostics.SectionResult{"pods": podsResult, "logs": logsResult} {
		b, err := json.Marshal(r)
		if err != nil {
			return manifest, nil, err
		}
		sections[name] = b
	}
	return manifest, sections, nil
}

func init() {
	supportCmd.AddCommand(supportBundleCmd)
	f := supportBundleCmd.Flags()
	f.Bool("offline", false, "Collect pods, events and logs with kubectl instead of asking the server (use when the server is down)")
	f.String("namespace", "nudgebee", "Namespace of the Nudgebee install (--offline)")
	f.String("context", "", "kubeconfig context to use (--offline; default: current context)")
	f.StringSlice("sections", nil, "Sections to collect: pods,logs,broker,agents,database (default: all; server mode)")
	f.String("case-id", "", "Support ticket number to record in the bundle")
	f.String("output-dir", ".", "Directory to write the bundle to")
	f.String("recipient", "", "age public key to encrypt to (default: the server's configured support key)")
}
