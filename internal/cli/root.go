package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "openhook",
	Short: "Webhook relay service for AI agents",
	Long:  "openhook - Webhook relay service enabling AI agents to subscribe to external platform events.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
