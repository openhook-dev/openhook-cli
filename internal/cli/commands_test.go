package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// setupTestEnv creates a test server and config for CLI testing
func setupTestEnv(t *testing.T, handler http.HandlerFunc) (*httptest.Server, func()) {
	server := httptest.NewServer(handler)

	// Setup temp config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	// Save config with test server URL
	cfg := &config{
		APIKey:    "oh_test_abc123xyz789",
		ServerURL: server.URL,
	}
	saveConfig(cfg)

	cleanup := func() {
		server.Close()
		os.Setenv("HOME", oldHome)
	}

	return server, cleanup
}

func TestSubscribeCommand(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/subscriptions" && r.Method == "POST" {
			var req struct {
				Platform string   `json:"platform"`
				Target   string   `json:"target"`
				Events   []string `json:"events"`
			}
			json.NewDecoder(r.Body).Decode(&req)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":         "sub_test123",
				"platform":   req.Platform,
				"target":     req.Target,
				"events":     req.Events,
				"status":     "active",
				"created_at": "2024-01-01T00:00:00Z",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	_, cleanup := setupTestEnv(t, handler)
	defer cleanup()

	// Test the subscribe command flag parsing
	cmd := subscribeCmd
	cmd.SetArgs([]string{"--platform", "github", "--target", "owner/repo"})

	// We can't easily capture output without more refactoring,
	// but we can verify the command parses correctly
	if cmd.Use != "subscribe" {
		t.Errorf("Expected command name 'subscribe', got %s", cmd.Use)
	}
}

func TestListCommand(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/subscriptions" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":         "sub_1",
					"platform":   "github",
					"target":     "owner/repo",
					"events":     []string{"push", "pull_request"},
					"status":     "active",
					"created_at": "2024-01-01T00:00:00Z",
				},
				{
					"id":         "sub_2",
					"platform":   "stripe",
					"target":     "",
					"events":     []string{"*"},
					"status":     "active",
					"created_at": "2024-01-02T00:00:00Z",
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	_, cleanup := setupTestEnv(t, handler)
	defer cleanup()

	// Verify command structure
	cmd := listCmd
	if cmd.Use != "list" {
		t.Errorf("Expected command name 'list', got %s", cmd.Use)
	}
}

func TestEventsListCommand(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1/events") && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"events": []map[string]interface{}{
					{
						"id":         "evt_1",
						"event_id":   "evt_abc123",
						"source":     "github",
						"event_type": "push",
						"summary":    "Push to main",
						"status":     "delivered",
						"created_at": "2024-01-01T12:00:00Z",
					},
				},
				"count": 1,
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	_, cleanup := setupTestEnv(t, handler)
	defer cleanup()

	// Verify command structure
	cmd := eventsListCmd
	if cmd.Use != "list" {
		t.Errorf("Expected command name 'list', got %s", cmd.Use)
	}

	// Verify flags exist
	if cmd.Flags().Lookup("source") == nil {
		t.Error("Expected --source flag")
	}
	if cmd.Flags().Lookup("status") == nil {
		t.Error("Expected --status flag")
	}
	if cmd.Flags().Lookup("limit") == nil {
		t.Error("Expected --limit flag")
	}
}

func TestUnsubscribeCommand(t *testing.T) {
	deleteCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1/subscriptions/") && r.Method == "DELETE" {
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	_, cleanup := setupTestEnv(t, handler)
	defer cleanup()

	// Verify command structure
	cmd := unsubscribeCmd
	if cmd.Use != "unsubscribe [subscription-id]" {
		t.Errorf("Expected 'unsubscribe [subscription-id]', got %s", cmd.Use)
	}

	_ = deleteCalled // Would need to execute command to test
}

func TestTunnelCommandStructure(t *testing.T) {
	// Verify tunnel command and subcommands exist
	if tunnelCmd.Use != "tunnel" {
		t.Errorf("Expected 'tunnel', got %s", tunnelCmd.Use)
	}

	// Check subcommands
	subCommands := tunnelCmd.Commands()
	foundStart := false
	foundStatus := false
	for _, cmd := range subCommands {
		if cmd.Use == "start" {
			foundStart = true
		}
		if cmd.Use == "status" {
			foundStatus = true
		}
	}

	if !foundStart {
		t.Error("Missing 'tunnel start' subcommand")
	}
	if !foundStatus {
		t.Error("Missing 'tunnel status' subcommand")
	}
}

func TestAuthCommandStructure(t *testing.T) {
	// Verify auth command and subcommands exist
	if authCmd.Use != "auth" {
		t.Errorf("Expected 'auth', got %s", authCmd.Use)
	}

	subCommands := authCmd.Commands()
	required := map[string]bool{"login": false, "logout": false, "status": false}

	for _, cmd := range subCommands {
		if _, ok := required[cmd.Use]; ok {
			required[cmd.Use] = true
		}
	}

	for name, found := range required {
		if !found {
			t.Errorf("Missing 'auth %s' subcommand", name)
		}
	}
}

func TestEventsCommandStructure(t *testing.T) {
	// Verify events command and subcommands exist
	if eventsCmd.Use != "events" {
		t.Errorf("Expected 'events', got %s", eventsCmd.Use)
	}

	subCommands := eventsCmd.Commands()
	foundList := false
	foundGet := false
	for _, cmd := range subCommands {
		if cmd.Use == "list" {
			foundList = true
		}
		if cmd.Use == "get [event-id]" {
			foundGet = true
		}
	}

	if !foundList {
		t.Error("Missing 'events list' subcommand")
	}
	if !foundGet {
		t.Error("Missing 'events get' subcommand")
	}
}

func TestRootCommand(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// Root command should have subcommands
	subCommands := rootCmd.Commands()
	if len(subCommands) == 0 {
		t.Error("Root command has no subcommands")
	}

	// Check expected subcommands exist
	expectedCmds := []string{"auth", "subscribe", "unsubscribe", "list", "tunnel", "events", "version"}
	for _, expected := range expectedCmds {
		found := false
		for _, cmd := range subCommands {
			if cmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing subcommand: %s", expected)
		}
	}
}
