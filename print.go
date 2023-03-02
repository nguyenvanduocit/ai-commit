package main

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
)

var normalStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63"))

var warningStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("202"))

var errorStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("9"))

var successStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("46"))

func printNormal(message string) {

	fmt.Println(normalStyle.
		SetString(message))
}

func printWarning(message string) {
	fmt.Println(warningStyle.
		SetString(message))
}

func printSuccess(message string) {
	fmt.Println(successStyle.
		SetString(message))
}

func printError(message string) {
	fmt.Println(errorStyle.
		SetString(message))

}
