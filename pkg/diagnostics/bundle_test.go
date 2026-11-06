package diagnostics

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
)

func readTarGz(t *testing.T, r io.Reader) map[string]string {
	t.Helper()
	gz, err := gzip.NewReader(r)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	out := map[string]string{}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(tr)
		out[h.Name] = string(b)
	}
}

func TestWriteEncryptedAndPlain(t *testing.T) {
	files, err := Files(Manifest{Format: FormatName, CaseID: "T-1"}, map[string]json.RawMessage{
		"pods": json.RawMessage(`{"status":"ok"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)

	path, encrypted, err := Write(t.TempDir(), files, id.Recipient().String(), now)
	if err != nil || !encrypted || !strings.HasSuffix(path, ".tar.gz.age") {
		t.Fatalf("path=%s encrypted=%v err=%v", path, encrypted, err)
	}
	f, _ := os.Open(path)
	defer func() { _ = f.Close() }()
	dec, err := age.Decrypt(f, id)
	if err != nil {
		t.Fatal(err)
	}
	got := readTarGz(t, dec)
	if !strings.Contains(got["manifest.json"], `"case_id": "T-1"`) || !strings.Contains(got["pods.json"], `"status": "ok"`) {
		t.Errorf("contents = %v", got)
	}

	plain, encrypted, err := Write(t.TempDir(), files, "", now)
	if err != nil || encrypted || !strings.HasSuffix(plain, ".tar.gz") {
		t.Fatalf("plain=%s encrypted=%v err=%v", plain, encrypted, err)
	}

	if _, _, err := Write(t.TempDir(), files, "not-a-key", now); err == nil {
		t.Fatal("expected invalid key error")
	}
}
