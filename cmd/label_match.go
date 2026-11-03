package cmd

import (
	"fmt"
	"regexp"
	"strings"
)

// labelMatcherArg splits a matcher at its first operator, so a value that itself
// contains "=" or "!=" stays part of the value.
var labelMatcherArg = regexp.MustCompile(`^([^=!~]*)(=~|!=|=)(.*)$`)

var labelMatcherOperators = map[string]string{"=": "_eq", "!=": "_neq", "=~": "_regex"}

// The part of the backend's refusal that names it: the account's provider cannot
// scope a value lookup. The values are still worth having, so the command lists
// them unscoped and says so, rather than failing and costing the caller a retry.
const (
	matchRefusedMetrics = "label_matchers are not supported"
	matchRefusedLogs    = "current_filters are not supported"
	matchRefusedWarning = "Warning: this provider cannot filter by --match, so these are the label's values everywhere."
)

// labelMatcher is one --match flag: only values from series or streams that
// also carry this label are listed.
type labelMatcher struct {
	Label    string
	Operator string // _eq, _neq or _regex, as the backend's label_matchers take them
	Value    string
}

// parseLabelMatchers reads repeatable --match flags: label=value,
// label!=value or label=~regex. Each flag holds one matcher, not a {selector}.
func parseLabelMatchers(args []string) ([]labelMatcher, error) {
	out := make([]labelMatcher, 0, len(args))
	for _, arg := range args {
		var m labelMatcher
		if parts := labelMatcherArg.FindStringSubmatch(arg); parts != nil {
			m = labelMatcher{Label: parts[1], Operator: labelMatcherOperators[parts[2]], Value: parts[3]}
		}
		m.Label = strings.TrimSpace(m.Label)
		m.Value = strings.Trim(strings.TrimSpace(m.Value), `"'`)
		if m.Operator == "" || m.Label == "" || m.Value == "" || strings.ContainsAny(m.Label, "{}\"' ,") || strings.Contains(m.Value, `"`) {
			return nil, fmt.Errorf("--match %q: want one label=value, label!=value or label=~regex per flag, e.g. --match namespace=demo --match pod=~api-.*", arg)
		}
		out = append(out, m)
	}
	return out, nil
}
