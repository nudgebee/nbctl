package cmd

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
)

func printNubiArtTo(out io.Writer) {
	myFigure := figure.NewFigure("NuBi", "", true)
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	for _, line := range myFigure.Slicify() {
		_, _ = fmt.Fprintln(out, yellow.Render(line))
	}
	_, _ = fmt.Fprintln(out)
}

func printNubiHelpTo(out io.Writer) {
	_, _ = fmt.Fprintln(out, "Welcome to NuBi, your Nudgebee AI assistant.")
	_, _ = fmt.Fprintln(out, "Here are some things you can do to get started:")
	_, _ = fmt.Fprintln(out, "- Type '/help' to see a list of available commands.")
	_, _ = fmt.Fprintln(out, "- Ask questions about your clusters, such as 'show me all high-priority alerts'.")
	_, _ = fmt.Fprintln(out, "- Use '/bookmarks' to save and manage important conversations.")
	_, _ = fmt.Fprintln(out)
}
