package diagnostics

import (
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	cases := []struct {
		in       string
		secret   string
		mustKeep string
	}{
		{`connecting with password=hunter2xyz to db`, "hunter2xyz", "connecting with password="},
		{`{"api_key": "abc123def456", "user": "bob"}`, "abc123def456", `"user": "bob"`},
		{`DB_PASSWORD: s3cretvalue`, "s3cretvalue", "DB_PASSWORD"},
		{`GET /x?access_token=tok123456&page=2`, "tok123456", "page=2"},
		{`Authorization: Bearer abcdefghijklmnop`, "abcdefghijklmnop", "Authorization"},
		{`dial postgres://admin:pa55word@db.example.com:5432/app`, "pa55word", "db.example.com:5432/app"},
		{`token eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U`, "eyJhbGciOiJIUzI1NiJ9", ""},
		{`aws key AKIAABCDEFGHIJKLMNOP used`, "AKIAABCDEFGHIJKLMNOP", "used"},
		{`llm call with sk-abcdefghijklmnopqrstu failed`, "sk-abcdefghijklmnopqrstu", "failed"},
		{`NUDGEBEE_LICENSE=eyJxxxxxxxxxx`, "eyJxxxxxxxxxx", "NUDGEBEE_LICENSE="},
		// JSON logged inside a JSON string, as structured loggers log bodies.
		{`{"level":"error","msg":"bad body {\"password\":\"hunter2xyz\"}"}`, "hunter2xyz", `"level":"error"`},
		{`{\"api_key\":\"abc123def\",\"user\":\"bob\"}`, "abc123def", `\"user\":\"bob\"`},
		{`password: "has \" quote secretvalue"`, "secretvalue", "password:"},
		{`Cookie: session=abc123; other=def456zz`, "def456zz", "Cookie:"},
		{`posting to https://hooks.slack.com/services/T000/B000/XXXXXXXXXXXX failed`, "XXXXXXXXXXXX", "failed"},
		{`token xoxb-123456789012-abcdefghij used`, "xoxb-123456789012", "used"},
		{`clone with ghp_abcdefghijklmnopqrstuvwxyz0123456789 ok`, "ghp_abcdefghij", "ok"},
		{`pat github_pat_11ABCDEFG0123456789_abcdefghijk set`, "github_pat_11ABCDEFG", "set"},
		{`maps key AIzaSyA1234567890abcdefghijklmnopqrstuv rejected`, "AIzaSyA1234567890", "rejected"},
	}
	for _, c := range cases {
		got := Redact(c.in)
		if strings.Contains(got, c.secret) {
			t.Errorf("Redact(%q) = %q, still contains %q", c.in, got, c.secret)
		}
		if c.mustKeep != "" && !strings.Contains(got, c.mustKeep) {
			t.Errorf("Redact(%q) = %q, lost %q", c.in, got, c.mustKeep)
		}
	}
}

func TestRedactLeavesPlainLinesAlone(t *testing.T) {
	for _, s := range []string{
		`level=error msg="queue consumer stopped" queue=notifications`,
		`routing key notifications.created dropped`,
		`processed 42 events in 3.1s`,
	} {
		if got := Redact(s); got != s {
			t.Errorf("Redact(%q) changed it to %q", s, got)
		}
	}
}

func TestNewRedactorRejectsMalformed(t *testing.T) {
	if _, err := NewRedactor("key\tpass\textra"); err == nil {
		t.Fatal("expected error for malformed key rule")
	}
	if _, err := NewRedactor("value\t(unclosed\tx"); err == nil {
		t.Fatal("expected error for bad regex")
	}
}
