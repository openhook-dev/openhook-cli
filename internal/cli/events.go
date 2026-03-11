package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "View webhook event history",
}

var eventsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent webhook events",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL, _ := cmd.Flags().GetString("server")
		client, err := newClientWithServer(serverURL)
		if err != nil {
			return err
		}

		source, _ := cmd.Flags().GetString("source")
		status, _ := cmd.Flags().GetString("status")
		limit, _ := cmd.Flags().GetInt("limit")

		path := "/api/v1/events?"
		if source != "" {
			path += "source=" + source + "&"
		}
		if status != "" {
			path += "status=" + status + "&"
		}
		if limit > 0 {
			path += fmt.Sprintf("limit=%d&", limit)
		}

		var response struct {
			Events []struct {
				ID          string  `json:"id"`
				EventID     string  `json:"event_id"`
				Source      string  `json:"source"`
				EventType   string  `json:"event_type"`
				Summary     string  `json:"summary"`
				Status      string  `json:"status"`
				DeliveredAt *string `json:"delivered_at"`
				CreatedAt   string  `json:"created_at"`
			} `json:"events"`
			Count int `json:"count"`
		}

		if err := client.get(path, &response); err != nil {
			return err
		}

		if response.Count == 0 {
			fmt.Println("No events found")
			return nil
		}

		fmt.Printf("Found %d events:\n\n", response.Count)

		for _, e := range response.Events {
			createdAt, _ := time.Parse(time.RFC3339, e.CreatedAt)
			timestamp := createdAt.Local().Format("2006-01-02 15:04:05")

			statusIcon := "○"
			switch e.Status {
			case "delivered":
				statusIcon = "✓"
			case "dropped":
				statusIcon = "✗"
			case "failed":
				statusIcon = "!"
			}

			fmt.Printf("[%s] %s %s/%s\n", timestamp, statusIcon, e.Source, e.EventType)
			fmt.Printf("         %s\n", e.Summary)
			fmt.Printf("         ID: %s  Status: %s\n\n", e.ID[:8], e.Status)
		}

		return nil
	},
}

var eventsGetCmd = &cobra.Command{
	Use:   "get [event-id]",
	Short: "Get details of a specific event",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL, _ := cmd.Flags().GetString("server")
		client, err := newClientWithServer(serverURL)
		if err != nil {
			return err
		}

		var event struct {
			ID          string  `json:"id"`
			EventID     string  `json:"event_id"`
			Source      string  `json:"source"`
			EventType   string  `json:"event_type"`
			Summary     string  `json:"summary"`
			Payload     any     `json:"payload"`
			Status      string  `json:"status"`
			DeliveredAt *string `json:"delivered_at"`
			CreatedAt   string  `json:"created_at"`
		}

		if err := client.get("/api/v1/events/"+args[0], &event); err != nil {
			return err
		}

		fmt.Printf("Event: %s\n", event.ID)
		fmt.Printf("Source: %s\n", event.Source)
		fmt.Printf("Type: %s\n", event.EventType)
		fmt.Printf("Summary: %s\n", event.Summary)
		fmt.Printf("Status: %s\n", event.Status)
		fmt.Printf("Created: %s\n", event.CreatedAt)
		if event.DeliveredAt != nil {
			fmt.Printf("Delivered: %s\n", *event.DeliveredAt)
		}

		if event.Payload != nil {
			fmt.Printf("\nPayload:\n%v\n", event.Payload)
		}

		return nil
	},
}

func init() {
	eventsListCmd.Flags().String("server", "", "Server URL")
	eventsListCmd.Flags().String("source", "", "Filter by source (github, stripe, linear)")
	eventsListCmd.Flags().String("status", "", "Filter by status (received, delivered, dropped)")
	eventsListCmd.Flags().Int("limit", 20, "Number of events to show")

	eventsGetCmd.Flags().String("server", "", "Server URL")

	eventsCmd.AddCommand(eventsListCmd)
	eventsCmd.AddCommand(eventsGetCmd)
	rootCmd.AddCommand(eventsCmd)
}
