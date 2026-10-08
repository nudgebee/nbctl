package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/nudgebee/nbctl/pkg/format"
	"github.com/spf13/cobra"
)

// printRows prints a list result. An empty result prints nothing in text mode
// and [] in JSON mode, and explains itself with emptyMsg on stderr, so callers
// (people and scripts alike) can tell "nothing found" from a silent failure.
func printRows(cmd *cobra.Command, table format.TabularData, count int, emptyMsg string) error {
	if count > 0 {
		format.GetFormat().Print(table)
		return nil
	}
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), emptyMsg)
	if format.GetFormat().Get() == "json" {
		return format.GetFormat().PrintRawJSON(json.RawMessage("[]"))
	}
	return nil
}
