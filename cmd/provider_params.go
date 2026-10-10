package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// addParamFlag adds a repeatable --param key=value flag for provider-specific
// parameters (e.g. Elasticsearch query_type=kql, a CloudWatch log group).
func addParamFlag(c *cobra.Command) {
	c.Flags().StringArray("param", nil, "Provider-specific parameter key=value, passed to the provider as a string (repeatable); providers ignore keys they don't use")
}

// providerParams builds the nested `request` map that providers read their own
// parameters from. It combines --param key=value pairs with named flags
// (key → value; empty values are skipped). Values are strings. It returns nil
// when nothing is set, so the request is unchanged for providers that need none.
func providerParams(cmd *cobra.Command, named map[string]string) (map[string]any, error) {
	params := map[string]any{}
	pairs, _ := cmd.Flags().GetStringArray("param")
	for _, pair := range pairs {
		key, value, ok := strings.Cut(pair, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid --param %q: want key=value", pair)
		}
		if _, dup := params[key]; dup {
			return nil, fmt.Errorf("--param %q given more than once", key)
		}
		params[key] = value
	}

	keys := make([]string, 0, len(named))
	for key := range named {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := named[key]
		if value == "" {
			continue
		}
		if _, dup := params[key]; dup {
			return nil, fmt.Errorf("%q is set by both --param and its own flag; use one", key)
		}
		params[key] = value
	}

	if len(params) == 0 {
		return nil, nil
	}
	return params, nil
}
