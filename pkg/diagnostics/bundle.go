package diagnostics

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"filippo.io/age"
)

// FormatName matches the server's bundle manifest, so support opens a bundle
// from the app and one from nbctl the same way:
//
//	age -d -i support.key nudgebee-diagnostics-<ts>.tar.gz.age | tar xz
const FormatName = "nudgebee-diagnostics/v1"

// SectionResult is one section file in the bundle.
type SectionResult struct {
	Status     string `json:"status"` // ok | error
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"duration_ms"`
	Data       any    `json:"data,omitempty"`
}

// Manifest is manifest.json in the bundle.
type Manifest struct {
	Format      string    `json:"format"`
	GeneratedAt time.Time `json:"generated_at"`
	GeneratedBy string    `json:"generated_by"`
	CaseID      string    `json:"case_id,omitempty"`
	Sections    []string  `json:"sections"`
	LogWindow   string    `json:"log_window"`
	Note        string    `json:"note"`
}

// File is one entry of the archive.
type File struct {
	Name    string
	Content []byte
}

// Files returns manifest.json plus one <section>.json per section, sorted.
func Files(manifest any, sections map[string]json.RawMessage) ([]File, error) {
	m, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	files := []File{{Name: "manifest.json", Content: m}}
	names := make([]string, 0, len(sections))
	for n := range sections {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, sections[n], "", "  "); err != nil {
			return nil, fmt.Errorf("section %s: %w", n, err)
		}
		files = append(files, File{Name: n + ".json", Content: pretty.Bytes()})
	}
	return files, nil
}

// Write packages files as tar.gz, encrypted to recipient when it is set, into
// dir. It returns the written path and whether it is encrypted.
func Write(dir string, files []File, recipient string, now time.Time) (string, bool, error) {
	var tgz bytes.Buffer
	gz := gzip.NewWriter(&tgz)
	tw := tar.NewWriter(gz)
	for _, f := range files {
		if err := tw.WriteHeader(&tar.Header{Name: f.Name, Mode: 0o644, Size: int64(len(f.Content)), ModTime: now, Format: tar.FormatUSTAR}); err != nil {
			return "", false, err
		}
		if _, err := tw.Write(f.Content); err != nil {
			return "", false, err
		}
	}
	if err := tw.Close(); err != nil {
		return "", false, err
	}
	if err := gz.Close(); err != nil {
		return "", false, err
	}

	stamp := strings.NewReplacer(":", "-", ".", "-").Replace(now.UTC().Format(time.RFC3339))
	name := filepath.Join(dir, "nudgebee-diagnostics-"+stamp+".tar.gz")
	recipient = strings.TrimSpace(recipient)
	if recipient == "" {
		return name, false, writeNew(name, tgz.Bytes())
	}

	r, err := age.ParseX25519Recipient(recipient)
	if err != nil {
		return "", false, fmt.Errorf("invalid support key %q: %w", recipient, err)
	}
	var enc bytes.Buffer
	w, err := age.Encrypt(&enc, r)
	if err != nil {
		return "", false, err
	}
	if _, err := io.Copy(w, &tgz); err != nil {
		return "", false, err
	}
	if err := w.Close(); err != nil {
		return "", false, err
	}
	name += ".age"
	return name, true, writeNew(name, enc.Bytes())
}

// writeNew creates name owner-readable only and never overwrites a file.
func writeNew(name string, data []byte) error {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
