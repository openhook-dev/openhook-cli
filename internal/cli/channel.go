package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// Channel represents a channel
type Channel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// ChannelSubscription represents a channel subscription
type ChannelSubscription struct {
	ID           string `json:"id"`
	ChannelID    string `json:"channel_id"`
	EndpointID   string `json:"endpoint_id"`
	EndpointName string `json:"endpoint_name"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
}

// ChannelMessage represents a channel message
type ChannelMessage struct {
	ID          string                 `json:"id"`
	ChannelID   string                 `json:"channel_id"`
	Content     string                 `json:"content"`
	Destination string                 `json:"destination"`
	SenderName  string                 `json:"sender_name,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   string                 `json:"created_at"`
}

// Channel API methods
func (c *apiClient) CreateChannel(name, description string) (*Channel, error) {
	req := struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}{name, description}

	var ch Channel
	if err := c.post("/api/v1/channels", req, &ch); err != nil {
		return nil, err
	}
	return &ch, nil
}

func (c *apiClient) ListChannels() ([]Channel, error) {
	var channels []Channel
	if err := c.get("/api/v1/channels", &channels); err != nil {
		return nil, err
	}
	return channels, nil
}

func (c *apiClient) GetChannel(id string) (*Channel, error) {
	var ch Channel
	if err := c.get("/api/v1/channels/"+id, &ch); err != nil {
		return nil, err
	}
	return &ch, nil
}

func (c *apiClient) CreateChannelSubscription(channelID, endpointID, name string) (*ChannelSubscription, error) {
	req := struct {
		EndpointID string `json:"endpoint_id"`
		Name       string `json:"name"`
	}{endpointID, name}

	var sub ChannelSubscription
	if err := c.post("/api/v1/channels/"+channelID+"/subscriptions", req, &sub); err != nil {
		return nil, err
	}
	return &sub, nil
}

func (c *apiClient) ListChannelSubscriptions(channelID string) ([]ChannelSubscription, error) {
	var subs []ChannelSubscription
	if err := c.get("/api/v1/channels/"+channelID+"/subscriptions", &subs); err != nil {
		return nil, err
	}
	return subs, nil
}

func (c *apiClient) DeleteChannelSubscription(channelID, name string) error {
	return c.delete("/api/v1/channels/" + channelID + "/subscriptions/" + name)
}

func (c *apiClient) SendChannelMessage(channelID, content, destination, senderName string) (*ChannelMessage, error) {
	req := struct {
		Content     string `json:"content"`
		Destination string `json:"destination"`
		SenderName  string `json:"sender_name,omitempty"`
	}{content, destination, senderName}

	var msg ChannelMessage
	if err := c.post("/api/v1/channels/"+channelID+"/messages", req, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// Commands
var channelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Manage channels for agent-to-agent communication",
}

var channelCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new channel",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL, _ := cmd.Flags().GetString("server")
		client, err := newClientWithServer(serverURL)
		if err != nil {
			return err
		}

		name := args[0]
		description, _ := cmd.Flags().GetString("description")

		ch, err := client.CreateChannel(name, description)
		if err != nil {
			return err
		}

		fmt.Printf("Channel created: %s\n", ch.ID)
		fmt.Printf("  Name: %s\n", ch.Name)
		if ch.Description != "" {
			fmt.Printf("  Description: %s\n", ch.Description)
		}

		return nil
	},
}

var channelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your channels",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL, _ := cmd.Flags().GetString("server")
		client, err := newClientWithServer(serverURL)
		if err != nil {
			return err
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")

		channels, err := client.ListChannels()
		if err != nil {
			return err
		}

		if jsonOutput {
			output, _ := json.MarshalIndent(channels, "", "  ")
			fmt.Println(string(output))
			return nil
		}

		if len(channels) == 0 {
			fmt.Println("No channels found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION\tSTATUS\tCREATED")
		for _, ch := range channels {
			desc := ch.Description
			if len(desc) > 30 {
				desc = desc[:27] + "..."
			}
			created := ch.CreatedAt
			if len(created) > 10 {
				created = created[:10]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				ch.ID, ch.Name, desc, ch.Status, created)
		}
		w.Flush()

		return nil
	},
}

var channelSubscribeCmd = &cobra.Command{
	Use:   "subscribe <channel>",
	Short: "Subscribe an endpoint to a channel",
	Long: `Subscribe an endpoint to a channel with a unique name.

The name is used to address this endpoint when sending messages.
Example: openhook channel subscribe deploy-team --endpoint ep_xxx --name deployer`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL, _ := cmd.Flags().GetString("server")
		client, err := newClientWithServer(serverURL)
		if err != nil {
			return err
		}

		channelID := args[0]
		endpointID, _ := cmd.Flags().GetString("endpoint")
		name, _ := cmd.Flags().GetString("name")

		if endpointID == "" {
			return fmt.Errorf("--endpoint is required")
		}
		if name == "" {
			return fmt.Errorf("--name is required")
		}

		sub, err := client.CreateChannelSubscription(channelID, endpointID, name)
		if err != nil {
			return err
		}

		fmt.Printf("Subscribed to channel\n")
		fmt.Printf("  Channel:  %s\n", sub.ChannelID)
		fmt.Printf("  Name:     %s\n", sub.Name)
		fmt.Printf("  Endpoint: %s (%s)\n", sub.EndpointName, sub.EndpointID)

		return nil
	},
}

var channelUnsubscribeCmd = &cobra.Command{
	Use:   "unsubscribe <channel>",
	Short: "Unsubscribe from a channel",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL, _ := cmd.Flags().GetString("server")
		client, err := newClientWithServer(serverURL)
		if err != nil {
			return err
		}

		channelID := args[0]
		name, _ := cmd.Flags().GetString("name")

		if name == "" {
			return fmt.Errorf("--name is required")
		}

		if err := client.DeleteChannelSubscription(channelID, name); err != nil {
			return err
		}

		fmt.Printf("Unsubscribed '%s' from channel %s\n", name, channelID)
		return nil
	},
}

var channelMembersCmd = &cobra.Command{
	Use:   "members <channel>",
	Short: "List members (subscriptions) in a channel",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL, _ := cmd.Flags().GetString("server")
		client, err := newClientWithServer(serverURL)
		if err != nil {
			return err
		}

		channelID := args[0]
		jsonOutput, _ := cmd.Flags().GetBool("json")

		subs, err := client.ListChannelSubscriptions(channelID)
		if err != nil {
			return err
		}

		if jsonOutput {
			output, _ := json.MarshalIndent(subs, "", "  ")
			fmt.Println(string(output))
			return nil
		}

		if len(subs) == 0 {
			fmt.Println("No members in this channel")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tENDPOINT\tSTATUS\tCREATED")
		for _, sub := range subs {
			created := sub.CreatedAt
			if len(created) > 10 {
				created = created[:10]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				sub.Name, sub.EndpointName, sub.Status, created)
		}
		w.Flush()

		return nil
	},
}

var channelSendCmd = &cobra.Command{
	Use:   "send <channel> <message>",
	Short: "Send a message to a channel",
	Long: `Send a message to a channel.

Use --to to specify the destination:
  --to deployer    Send only to 'deployer'
  --to all         Send to all members (fan-out)

Examples:
  openhook channel send deploy-team "ship v2.1" --to deployer
  openhook channel send deploy-team "v2.1 is live" --to all`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL, _ := cmd.Flags().GetString("server")
		client, err := newClientWithServer(serverURL)
		if err != nil {
			return err
		}

		channelID := args[0]
		content := args[1]
		destination, _ := cmd.Flags().GetString("to")
		senderName, _ := cmd.Flags().GetString("from")

		if destination == "" {
			return fmt.Errorf("--to is required (member name or 'all')")
		}

		msg, err := client.SendChannelMessage(channelID, content, destination, senderName)
		if err != nil {
			return err
		}

		fmt.Printf("Message sent: %s\n", msg.ID)
		fmt.Printf("  To: %s\n", msg.Destination)
		if msg.SenderName != "" {
			fmt.Printf("  From: %s\n", msg.SenderName)
		}
		fmt.Printf("  Content: %s\n", truncate(msg.Content, 50))

		return nil
	},
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func init() {
	// channel create
	channelCreateCmd.Flags().String("description", "", "Channel description")
	channelCreateCmd.Flags().String("server", "", "Server URL")

	// channel list
	channelListCmd.Flags().Bool("json", false, "Output as JSON")
	channelListCmd.Flags().String("server", "", "Server URL")

	// channel subscribe
	channelSubscribeCmd.Flags().String("endpoint", "", "Endpoint ID to subscribe")
	channelSubscribeCmd.Flags().String("name", "", "Unique name for this subscription in the channel")
	channelSubscribeCmd.Flags().String("server", "", "Server URL")

	// channel unsubscribe
	channelUnsubscribeCmd.Flags().String("name", "", "Name of the subscription to remove")
	channelUnsubscribeCmd.Flags().String("server", "", "Server URL")

	// channel members
	channelMembersCmd.Flags().Bool("json", false, "Output as JSON")
	channelMembersCmd.Flags().String("server", "", "Server URL")

	// channel send
	channelSendCmd.Flags().String("to", "", "Destination: member name or 'all'")
	channelSendCmd.Flags().String("from", "", "Sender name (optional, for context)")
	channelSendCmd.Flags().String("server", "", "Server URL")

	channelCmd.AddCommand(channelCreateCmd)
	channelCmd.AddCommand(channelListCmd)
	channelCmd.AddCommand(channelSubscribeCmd)
	channelCmd.AddCommand(channelUnsubscribeCmd)
	channelCmd.AddCommand(channelMembersCmd)
	channelCmd.AddCommand(channelSendCmd)
	rootCmd.AddCommand(channelCmd)
}
