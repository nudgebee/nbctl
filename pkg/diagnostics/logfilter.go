// Ported from the Nudgebee server (api-server/services/diagnostics) so an
// offline bundle filters and masks logs exactly like the in-app one.
// redact_patterns.txt is a verbatim copy, checked against the server's in CI.
package diagnostics

import (
	"encoding/json"
	"regexp"
	"strings"
)

// Text-format level markers: slog text (level=ERROR), Python logging
// ("ERROR:" / " ERROR "), klog (E1105 ...), panics and tracebacks.
var textProblemLine = regexp.MustCompile(`(?i)(\blevel=(error|warn|warning|fatal|panic|critical)\b|\b(ERROR|WARN|WARNING|FATAL|CRITICAL|PANIC)\b|^[EWF]\d{4} |^panic:|^Traceback \(most recent call last\)|^[A-Za-z_.]*(Error|Exception): |\bException\b)`)

// Continuation lines of a stack trace or multi-line message.
var continuationLine = regexp.MustCompile(`^(\s+|goroutine \d+|Caused by:|\t)`)

var problemLevels = map[string]bool{
	"error": true, "err": true, "warn": true, "warning": true,
	"fatal": true, "panic": true, "critical": true, "crit": true,
}

func isProblemLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "{") {
		var rec map[string]any
		if json.Unmarshal([]byte(trimmed), &rec) == nil {
			for _, k := range []string{"level", "severity", "lvl", "levelname"} {
				if v, ok := rec[k].(string); ok {
					return problemLevels[strings.ToLower(v)]
				}
			}
		}
	}
	return textProblemLine.MatchString(line)
}

// FilteredLog is what the bundle keeps from one container's log.
type FilteredLog struct {
	// Problems: error/warn lines plus the stack-trace lines that follow them,
	// newest kept when over the cap.
	Problems []string `json:"problems"`
	// Tail: the last lines of the log regardless of level.
	Tail []string `json:"tail"`
	// TotalLines read from the API (after the time/bytes limits).
	TotalLines int `json:"total_lines"`
	// ProblemsTruncated is true when older problem lines were dropped.
	ProblemsTruncated bool `json:"problems_truncated,omitempty"`
}

// FilterLog keeps problem lines (with their continuation lines) and the last
// tailN lines, redacted. Full logs never enter the bundle.
func FilterLog(raw string, tailN, maxProblems int) FilteredLog {
	// Redact the whole text once: a multi-line secret (a PEM block) only
	// matches as one string, and the line-anchored base64 rule catches the body
	// of a block whose BEGIN line fell outside the read window.
	raw = Redact(raw)
	lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	if raw == "" {
		lines = nil
	}
	out := FilteredLog{TotalLines: len(lines), Problems: []string{}, Tail: []string{}}

	inProblem := false
	for _, line := range lines {
		switch {
		case isProblemLine(line):
			inProblem = true
			out.Problems = append(out.Problems, line)
		case inProblem && continuationLine.MatchString(line):
			out.Problems = append(out.Problems, line)
		default:
			inProblem = false
		}
	}
	if len(out.Problems) > maxProblems {
		out.Problems = out.Problems[len(out.Problems)-maxProblems:]
		out.ProblemsTruncated = true
	}

	start := len(lines) - tailN
	if start < 0 {
		start = 0
	}
	out.Tail = append(out.Tail, lines[start:]...)

	return out
}
