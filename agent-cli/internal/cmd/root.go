package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "agent-cli",
	Short: "A CLI tool for generating Go agent projects",
	Long: `Go Agent CLI is a command-line tool for generating Go-based agent projects.
It supports two modes:
- default: Standard agent with skills system
- provider: Integration agent with external service adapters`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Go Agent CLI - Use 'agent-cli --help' for more information")
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(versionCmd)
}
