package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tsisar/alert-agent/internal/version"
)

var versionCmd = &cobra.Command{
	Use:               "version",
	Short:             "Print build version",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("alert-agent %s (commit %s, built %s)\n", version.Version, version.Commit, version.Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}