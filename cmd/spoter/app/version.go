package app

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	VERSION = "0.1.0"
)

var Build string

var versionCmd = &cobra.Command{
	Use:           "version",
	Short:         "Show the current version",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Version: %s\nBuild: %s\n", VERSION, Build)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
