package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type config struct {
	APIKey    string `json:"api_key"`
	ServerURL string `json:"server_url,omitempty"`
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openhook")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

func loadConfig() (*config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, err
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

func saveConfig(cfg *config) error {
	if err := os.MkdirAll(configDir(), 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(configPath(), data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func maskKey(key string) string {
	if len(key) <= 12 {
		return key
	}
	prefix := key[:8]
	suffix := key[len(key)-4:]
	return prefix + strings.Repeat("*", len(key)-12) + suffix
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with an API key",
	RunE: func(cmd *cobra.Command, args []string) error {
		key, _ := cmd.Flags().GetString("key")
		if key == "" {
			return fmt.Errorf("--key flag is required")
		}

		if !strings.HasPrefix(key, "oh_live_") && !strings.HasPrefix(key, "oh_test_") {
			return fmt.Errorf("invalid key format: must start with oh_live_ or oh_test_")
		}

		cfg := &config{APIKey: key}
		if err := saveConfig(cfg); err != nil {
			return err
		}

		fmt.Printf("Authenticated successfully (%s)\n", maskKey(key))
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil || cfg.APIKey == "" {
			fmt.Println("Not authenticated")
			os.Exit(1)
		}

		serverURL, _ := cmd.Flags().GetString("server")
		client, err := newClientWithServer(serverURL)
		if err != nil {
			fmt.Println("Not authenticated")
			os.Exit(1)
		}

		me, err := client.GetMe()
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		fmt.Printf("Authenticated as %s (%s)\n", me.Email, maskKey(cfg.APIKey))
		if len(me.ConnectedPlatforms) > 0 {
			fmt.Printf("Connected platforms: %s\n", strings.Join(me.ConnectedPlatforms, ", "))
		} else {
			fmt.Println("No connected platforms")
		}

		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := configPath()
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Println("Not authenticated")
			return nil
		}

		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove config: %w", err)
		}

		fmt.Println("Logged out successfully")
		return nil
	},
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the current authenticated user",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil || cfg.APIKey == "" {
			fmt.Println("Not authenticated")
			os.Exit(1)
		}

		serverURL, _ := cmd.Flags().GetString("server")
		client, err := newClientWithServer(serverURL)
		if err != nil {
			fmt.Println("Not authenticated")
			os.Exit(1)
		}

		me, err := client.GetMe()
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		fmt.Println(me.Email)
		return nil
	},
}

func init() {
	authLoginCmd.Flags().String("key", "", "API key (oh_live_xxx or oh_test_xxx)")
	authStatusCmd.Flags().String("server", "", "Server URL")
	whoamiCmd.Flags().String("server", "", "Server URL")

	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(whoamiCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)
}
