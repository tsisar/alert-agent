package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/tsisar/alert-agent/internal/config"
	"github.com/tsisar/extended-log-go/log"
)

var (
	cfgFile string
	cfg     config.Config
)

var rootCmd = &cobra.Command{
	Use:   "alert-agent",
	Short: "Grafana alert investigation agent",
	Long:  "Receives Grafana alerts via webhook, investigates them using LLM + MCP tools, and sends reports to Telegram/Slack.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return err
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Errorf("command failed: %v", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (default: ./configs/config.yaml)")
}
