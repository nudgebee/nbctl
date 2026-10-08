package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// logQueryTypes are the query languages logs query --query-type accepts.
// Elasticsearch reads them from the nested request map: dsl (default), kql
// (hosted Elasticsearch only) and ppl (OpenSearch).
var logQueryTypes = []string{"dsl", "kql", "ppl"}

// metricQueryTypes are the query languages metrics query --query-type accepts.
// Elasticsearch metrics take dsl or kql; without a type, --query must be
// Nudgebee's where-clause JSON.
var metricQueryTypes = []string{"dsl", "kql"}

// addParamFlag adds a repeatable --param key=value flag for provider-specific
// parameters (e.g. Elasticsearch index, CloudWatch log group).
func addParamFlag(c *cobra.Command) {
	c.Flags().StringArray("param", nil, "Provider-specific parameter key=value, sent in the nested request map (repeatable)")
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

func validateQueryType(queryType string, allowed []string) error {
	if queryType == "" {
		return nil
	}
	for _, t := range allowed {
		if queryType == t {
			return nil
		}
	}
	return fmt.Errorf("invalid --query-type %q: want one of %s", queryType, strings.Join(allowed, ", "))
}
