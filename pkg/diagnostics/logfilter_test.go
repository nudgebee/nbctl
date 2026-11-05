package diagnostics

import (
	"strings"
	"testing"
)

func TestFilterLogKeepsProblemsAndTraces(t *testing.T) {
	raw := strings.Join([]string{
		`{"time":"t1","level":"INFO","msg":"started"}`,
		`{"time":"t2","level":"ERROR","msg":"db down","password":"hunter2xyz"}`,
		`time=t3 level=INFO msg=ok`,
		`time=t4 level=WARN msg="slow query"`,
		`panic: runtime error: index out of range`,
		``,
		`goroutine 1 [running]:`,
		`	main.go:12 +0x1d`,
		`2026-11-05 10:00:00 INFO worker ready`,
		`Traceback (most recent call last):`,
		`  File "app.py", line 3, in <module>`,
		`ValueError: bad`,
		`E1105 10:00:01.000 relay.go:10] dial failed`,
		`processed 3 items`,
	}, "\n") + "\n"

	got := FilterLog(raw, 2, 100)

	if got.TotalLines != 14 {
		t.Errorf("TotalLines = %d, want 14", got.TotalLines)
	}
	joined := strings.Join(got.Problems, "\n")
	for _, want := range []string{"db down", "slow query", "panic: runtime error", "Traceback", `File "app.py"`, "ValueError: bad", "dial failed"} {
		if !strings.Contains(joined, want) {
			t.Errorf("problems missing %q:\n%s", want, joined)
		}
	}
	for _, unwanted := range []string{"started", "msg=ok", "worker ready", "processed 3 items", "hunter2xyz"} {
		if strings.Contains(joined, unwanted) {
			t.Errorf("problems should not contain %q:\n%s", unwanted, joined)
		}
	}
	if len(got.Tail) != 2 || got.Tail[1] != "processed 3 items" {
		t.Errorf("tail = %v", got.Tail)
	}
}

func TestFilterLogCapsProblemsKeepingNewest(t *testing.T) {
	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, "level=error msg=e"+string(rune('0'+i)))
	}
	got := FilterLog(strings.Join(lines, "\n"), 0, 3)
	if !got.ProblemsTruncated || len(got.Problems) != 3 || !strings.HasSuffix(got.Problems[2], "e9") {
		t.Fatalf("got %+v", got)
	}
}

func TestFilterLogEmpty(t *testing.T) {
	got := FilterLog("", 50, 50)
	if got.TotalLines != 0 || len(got.Problems) != 0 || len(got.Tail) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestFilterLogMasksPrivateKeyBlocks(t *testing.T) {
	body := "MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7VJTUt9Us8cKj"
	full := "level=info msg=loading key\n-----BEGIN PRIVATE KEY-----\n" + body + "\n-----END PRIVATE KEY-----\nlevel=info msg=done\n"
	// The second case starts mid-block: the BEGIN line fell outside the window.
	for _, raw := range []string{full, body + "\n-----END PRIVATE KEY-----\nlevel=info msg=done\n"} {
		got := FilterLog(raw, 10, 10)
		if strings.Contains(strings.Join(got.Tail, "\n"), body) {
			t.Errorf("key body leaked into tail: %v", got.Tail)
		}
	}
}
