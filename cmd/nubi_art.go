package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
)

func printNubiArt() {
	myFigure := figure.NewFigure("NuBi", "", true)
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	for _, line := range myFigure.Slicify() {
		fmt.Println(yellow.Render(line))
	}
	fmt.Println()
}

func printNubiHelp() {
	fmt.Println("Welcome to NuBi, your Nudgebee AI assistant.")
	fmt.Println("Here are some things you can do to get started:")
	fmt.Println("- Type '/help' to see a list of available commands.")
	fmt.Println("- Ask questions about your clusters, such as 'show me all high-priority alerts'.")
	fmt.Println("- Use '/bookmarks' to save and manage important conversations.")
	fmt.Println()
}
