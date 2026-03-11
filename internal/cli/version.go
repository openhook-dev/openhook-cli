package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "0.1.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the openhook version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("openhook %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
