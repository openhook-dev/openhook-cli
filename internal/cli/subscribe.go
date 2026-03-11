package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var subscribeCmd = &cobra.Command{
	Use:   "subscribe",
	Short: "Subscribe to platform events",
	Long:  "Subscribe to webhook events from connected platforms (github, stripe, linear)",
}

var subscribeGithubCmd = &cobra.Command{
	Use:   "github",
	Short: "Subscribe to GitHub repository events",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, _ := cmd.Flags().GetString("repo")
		events, _ := cmd.Flags().GetString("events")

		if repo == "" {
			return fmt.Errorf("--repo is required (format: owner/repo)")
		}
		if events == "" {
			return fmt.Errorf("--events is required (e.g., push,pull_request,workflow_run)")
		}
		if !strings.Contains(repo, "/") {
			return fmt.Errorf("invalid repo format, expected owner/repo")
		}

		return doSubscribe(cmd, "github", repo, events)
	},
}

var subscribeStripeCmd = &cobra.Command{
	Use:   "stripe",
	Short: "Subscribe to Stripe payment events",
	RunE: func(cmd *cobra.Command, args []string) error {
		events, _ := cmd.Flags().GetString("events")

		if events == "" {
			return fmt.Errorf("--events is required (e.g., payment_intent.failed,invoice.paid)")
		}

		return doSubscribe(cmd, "stripe", "account", events)
	},
}

var subscribeLinearCmd = &cobra.Command{
	Use:   "linear",
	Short: "Subscribe to Linear issue events",
	RunE: func(cmd *cobra.Command, args []string) error {
		team, _ := cmd.Flags().GetString("team")
		events, _ := cmd.Flags().GetString("events")

		if events == "" {
			return fmt.Errorf("--events is required (e.g., issue.created,issue.updated)")
		}

		target := "workspace"
		if team != "" {
			target = team
		}

		return doSubscribe(cmd, "linear", target, events)
	},
}

func doSubscribe(cmd *cobra.Command, platform, target, events string) error {
	serverURL, _ := cmd.Flags().GetString("server")
	client, err := newClientWithServer(serverURL)
	if err != nil {
		return err
	}

	eventList := parseEvents(events)

	sub, err := client.CreateSubscription(platform, target, eventList)
	if err != nil {
		return err
	}

	fmt.Printf("Subscription created: %s\n", sub.ID)
	fmt.Printf("  Platform: %s\n", sub.Platform)
	fmt.Printf("  Target:   %s\n", sub.Target)
	fmt.Printf("  Events:   %s\n", strings.Join(sub.Events, ", "))

	return nil
}

func parseEvents(events string) []string {
	parts := strings.Split(events, ",")
	result := make([]string, 0, len(parts))
	for _, e := range parts {
		if trimmed := strings.TrimSpace(e); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func init() {
	subscribeGithubCmd.Flags().String("repo", "", "Repository (format: owner/repo)")
	subscribeGithubCmd.Flags().String("events", "", "Events to subscribe to (comma-separated)")
	subscribeGithubCmd.Flags().String("server", "", "Server URL")

	subscribeStripeCmd.Flags().String("events", "", "Events to subscribe to (comma-separated)")
	subscribeStripeCmd.Flags().String("server", "", "Server URL")

	subscribeLinearCmd.Flags().String("team", "", "Team ID (optional)")
	subscribeLinearCmd.Flags().String("events", "", "Events to subscribe to (comma-separated)")
	subscribeLinearCmd.Flags().String("server", "", "Server URL")

	subscribeCmd.AddCommand(subscribeGithubCmd)
	subscribeCmd.AddCommand(subscribeStripeCmd)
	subscribeCmd.AddCommand(subscribeLinearCmd)
	rootCmd.AddCommand(subscribeCmd)
}
