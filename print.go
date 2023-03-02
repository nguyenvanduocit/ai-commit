package main

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
)

var normalStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63"))

var warningStyle = normalStyle.Copy().BorderForeground(lipgloss.Color("202"))

var errorStyle = normalStyle.Copy().BorderForeground(lipgloss.Color("9"))

var successStyle = normalStyle.Copy().BorderForeground(lipgloss.Color("46"))

func printNormal(message string) {

	fmt.Println(normalStyle.
		Render(message))
}

func printWarning(message string) {
	fmt.Println(warningStyle.
		Render(message))
}

func printSuccess(message string) {
	fmt.Println(successStyle.
		Render(message))
}

func printError(message string) {
	fmt.Println(errorStyle.
		Render(message))

}
