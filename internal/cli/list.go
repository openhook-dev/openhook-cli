package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List active subscriptions",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL, _ := cmd.Flags().GetString("server")
		client, err := newClientWithServer(serverURL)
		if err != nil {
			return err
		}

		platform, _ := cmd.Flags().GetString("platform")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		subs, err := client.ListSubscriptions(platform)
		if err != nil {
			return err
		}

		if jsonOutput {
			output, _ := json.MarshalIndent(subs, "", "  ")
			fmt.Println(string(output))
			return nil
		}

		if len(subs) == 0 {
			fmt.Println("No subscriptions found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tPLATFORM\tTARGET\tEVENTS\tSTATUS\tCREATED")
		for _, sub := range subs {
			events := strings.Join(sub.Events, ",")
			if len(events) > 30 {
				events = events[:27] + "..."
			}
			created := sub.CreatedAt
			if len(created) > 10 {
				created = created[:10]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				sub.ID, sub.Platform, sub.Target, events, sub.Status, created)
		}
		w.Flush()

		return nil
	},
}

func init() {
	listCmd.Flags().String("platform", "", "Filter by platform (github, stripe, linear)")
	listCmd.Flags().Bool("json", false, "Output as JSON")
	listCmd.Flags().String("server", "", "Server URL")
	rootCmd.AddCommand(listCmd)
}
