// Ported from the Nudgebee server (api-server/services/diagnostics) so an
// offline bundle filters and masks logs exactly like the in-app one.
// redact_patterns.txt is a verbatim copy, checked against the server's in CI.
package diagnostics

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"
)

//go:embed redact_patterns.txt
var redactPatternsFile string

type valueRule struct {
	re      *regexp.Regexp
	replace string
}

// Redactor masks secrets in bundle text. It is a backstop: the bundle already
// keeps only error/warn lines and a short tail, because no pattern list can
// recognise customer data inside a log line.
type Redactor struct {
	keyRule    *regexp.Regexp
	valueRules []valueRule
}

var defaultRedactor = mustRedactor(redactPatternsFile)

func mustRedactor(src string) *Redactor {
	r, err := NewRedactor(src)
	if err != nil {
		panic(err)
	}
	return r
}

// NewRedactor parses the rule format documented in redact_patterns.txt.
func NewRedactor(src string) (*Redactor, error) {
	var keys []string
	r := &Redactor{}
	for i, line := range strings.Split(src, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		switch {
		case fields[0] == "key" && len(fields) == 2:
			if _, err := regexp.Compile(fields[1]); err != nil {
				return nil, fmt.Errorf("redact rule line %d: %w", i+1, err)
			}
			keys = append(keys, fields[1])
		case fields[0] == "value" && len(fields) == 3:
			re, err := regexp.Compile(fields[1])
			if err != nil {
				return nil, fmt.Errorf("redact rule line %d: %w", i+1, err)
			}
			r.valueRules = append(r.valueRules, valueRule{re: re, replace: fields[2]})
		default:
			return nil, fmt.Errorf("redact rule line %d: malformed", i+1)
		}
	}
	if len(keys) > 0 {
		// Matches `password=x`, `"api_key": "x"`, `DB_PASSWORD: x`, `?token=x&`,
		// and JSON logged inside a JSON string (`{\"password\":\"x\"}`).
		// The key may carry a prefix/suffix (db_password, refresh_token).
		// Value forms, tried in order: an escaped-quoted string, a quoted
		// string that may contain \" , a single-quoted string, a bare token.
		r.keyRule = regexp.MustCompile(`(?i)(\\?"?[A-Za-z0-9_.-]*(?:` + strings.Join(keys, "|") +
			`)[A-Za-z0-9_.-]*\\?"?\s*[:=]\s*)(\\"(?:[^"\\]|\\[^"])*\\"|"(?:[^"\\]|\\.)*"|'[^']*'|[^\s,;&}"'\\]+)`)
	}
	return r, nil
}

// Redact returns s with every rule applied.
func (r *Redactor) Redact(s string) string {
	for _, rule := range r.valueRules {
		s = rule.re.ReplaceAllString(s, rule.replace)
	}
	if r.keyRule != nil {
		s = r.keyRule.ReplaceAllString(s, `${1}[REDACTED]`)
	}
	return s
}

// Redact applies the built-in rules.
func Redact(s string) string {
	return defaultRedactor.Redact(s)
}
