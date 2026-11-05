package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nudgebee/nbctl/pkg/diagnostics"
	"github.com/nudgebee/nbctl/pkg/testutil"
)

func TestSupportBundleServerMode(t *testing.T) {
	var gotBody string
	url, stop := testutil.StartMockServer(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"diagnostics_generate_bundle":{
			"manifest":{"format":"nudgebee-diagnostics/v1","case_id":"T-9"},
			"sections":{"agents":{"status":"ok","data":{"agents":[]}}},
			"support_public_key":""}}}`))
	})
	defer stop()
	restore := testutil.WithViper(map[string]any{"endpoint": url, "api-key": "sk-nb-test", "account-id": "acc-1"})
	defer restore()

	dir := t.TempDir()
	out, err := testutil.RunCommandCaptureOutput(supportBundleCmd,
		[]string{"support", "bundle", "--sections", "agents", "--case-id", "T-9", "--output-dir", dir, "--format", "json"})
	if err != nil {
		t.Fatalf("err=%v out=%s", err, out)
	}
	if !strings.Contains(gotBody, "diagnostics_generate_bundle") || !strings.Contains(gotBody, `"caseId":"T-9"`) {
		t.Errorf("request body = %s", gotBody)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "nudgebee-diagnostics-*.tar.gz"))
	if len(matches) != 1 {
		t.Fatalf("bundle files = %v (out=%s)", matches, out)
	}
	if !strings.Contains(out, `"mode": "server"`) && !strings.Contains(out, `"mode":"server"`) {
		t.Errorf("output = %s", out)
	}
}

func TestSupportBundleRejectsUnknownSection(t *testing.T) {
	restore := testutil.WithViper(map[string]any{"endpoint": "http://127.0.0.1:1", "api-key": "sk-nb-test", "account-id": "acc-1"})
	defer restore()
	_, err := testutil.RunCommandCaptureOutput(supportBundleCmd, []string{"support", "bundle", "--sections", "nope", "--output-dir", t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "unknown section") {
		t.Fatalf("err = %v", err)
	}
}

func TestCollectOffline(t *testing.T) {
	k := &diagnostics.Kubectl{Namespace: "nb", Run: func(_ context.Context, args ...string) ([]byte, error) {
		switch strings.Join(args[:2], " ") {
		case "get pods":
			return []byte(`{"items":[{"metadata":{"name":"p1"},"spec":{"containers":[{"name":"c1","image":"img:1"}]},
				"status":{"phase":"Running","containerStatuses":[{"name":"c1","ready":true,"restartCount":0,"state":{"running":{}}}]}}]}`), nil
		case "get events":
			return []byte(`{"items":[]}`), nil
		}
		return []byte("level=error msg=failed\n"), nil
	}}
	manifest, sections, err := collectOffline(context.Background(), k, "T-2", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if manifest.CaseID != "T-2" || manifest.GeneratedBy != "nbctl --offline" {
		t.Errorf("manifest = %+v", manifest)
	}
	var logs diagnostics.SectionResult
	_ = json.Unmarshal(sections["logs"], &logs)
	if logs.Status != "ok" || !strings.Contains(string(sections["logs"]), "msg=failed") {
		t.Errorf("logs = %s", sections["logs"])
	}
}

func TestSupportNotAnMCPTool(t *testing.T) {
	// registerCommands skips "support"; keep the skip list and the command name in step.
	src, err := os.ReadFile("mcp.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), `c.Name() == "support"`) {
		t.Fatal("support must not be exposed as an MCP tool")
	}
}

func TestSupportBundleServerModeNeedsConfig(t *testing.T) {
	restore := testutil.WithViper(map[string]any{"endpoint": "", "api-key": "", "account-id": ""})
	defer restore()
	_, err := testutil.RunCommandCaptureOutput(supportBundleCmd, []string{"support", "bundle", "--output-dir", t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "--offline") {
		t.Fatalf("err = %v", err)
	}
}
