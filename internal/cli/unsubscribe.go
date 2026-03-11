package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var unsubscribeCmd = &cobra.Command{
	Use:   "unsubscribe [subscription-id]",
	Short: "Remove a subscription",
	Long:  "Remove a subscription by ID, or use --all to remove all subscriptions for a platform",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL, _ := cmd.Flags().GetString("server")
		client, err := newClientWithServer(serverURL)
		if err != nil {
			return err
		}

		all, _ := cmd.Flags().GetBool("all")
		platform, _ := cmd.Flags().GetString("platform")
		yes, _ := cmd.Flags().GetBool("yes")

		if all {
			if platform == "" {
				return fmt.Errorf("--platform is required when using --all")
			}
			return unsubscribeAll(client, platform, yes)
		}

		if len(args) == 0 {
			return fmt.Errorf("subscription ID is required (or use --all --platform <platform>)")
		}

		subID := args[0]
		if !strings.HasPrefix(subID, "sub_") {
			return fmt.Errorf("invalid subscription ID format, expected sub_xxx")
		}

		if err := client.DeleteSubscription(subID); err != nil {
			return err
		}
		fmt.Printf("Unsubscribed: %s\n", subID)
		return nil
	},
}

func unsubscribeAll(client *apiClient, platform string, skipConfirm bool) error {
	subs, err := client.ListSubscriptions(platform)
	if err != nil {
		return err
	}

	if len(subs) == 0 {
		fmt.Printf("No %s subscriptions found\n", platform)
		return nil
	}

	if !skipConfirm {
		fmt.Printf("This will remove %d %s subscription(s):\n", len(subs), platform)
		for _, sub := range subs {
			fmt.Printf("  - %s (%s)\n", sub.ID, sub.Target)
		}
		fmt.Print("Continue? [y/N] ")

		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	deleted := 0
	for _, sub := range subs {
		if err := client.DeleteSubscription(sub.ID); err != nil {
			fmt.Printf("Failed to remove %s: %v\n", sub.ID, err)
		} else {
			fmt.Printf("Unsubscribed: %s\n", sub.ID)
			deleted++
		}
	}

	fmt.Printf("Removed %d subscription(s)\n", deleted)
	return nil
}

func init() {
	unsubscribeCmd.Flags().Bool("all", false, "Remove all subscriptions for a platform")
	unsubscribeCmd.Flags().String("platform", "", "Platform to filter (required with --all)")
	unsubscribeCmd.Flags().Bool("yes", false, "Skip confirmation prompt")
	unsubscribeCmd.Flags().String("server", "", "Server URL")
	rootCmd.AddCommand(unsubscribeCmd)
}
