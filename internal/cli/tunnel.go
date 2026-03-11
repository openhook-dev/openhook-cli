package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

// OpenClaw integration config
type openclawConfig struct {
	enabled bool
	url     string
	token   string
}

// tunnelConfig holds tunnel runtime configuration
type tunnelConfig struct {
	openclaw   *openclawConfig
	timeout    time.Duration
	maxEvents  int
	jsonOutput bool
}

const (
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
	backoffFactor  = 2
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Manage tunnel connections",
}

var tunnelStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a tunnel to receive webhook events",
	Long: `Start a tunnel to receive webhook events in real-time.

Use --openclaw to forward events to OpenClaw's /hooks/agent endpoint:
  openhook tunnel start --openclaw --openclaw-token $OPENCLAW_HOOKS_TOKEN

This will send each webhook event to your local OpenClaw instance,
allowing your AI agent to react to GitHub pushes, Stripe payments, etc.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("not authenticated, run: openhook auth login --key <your-key>")
		}

		serverURL, _ := cmd.Flags().GetString("server")
		if serverURL == "" && cfg.ServerURL != "" {
			serverURL = cfg.ServerURL
		}
		if serverURL == "" {
			serverURL = defaultServerURL
		}

		// OpenClaw config
		openclawEnabled, _ := cmd.Flags().GetBool("openclaw")
		openclawURL, _ := cmd.Flags().GetString("openclaw-url")
		openclawToken, _ := cmd.Flags().GetString("openclaw-token")

		// Check for token in env if not provided
		if openclawEnabled && openclawToken == "" {
			openclawToken = os.Getenv("OPENCLAW_HOOKS_TOKEN")
		}

		if openclawEnabled && openclawToken == "" {
			return fmt.Errorf("--openclaw-token required or set OPENCLAW_HOOKS_TOKEN env var")
		}

		// Timeout and max-events config
		timeoutStr, _ := cmd.Flags().GetString("timeout")
		maxEvents, _ := cmd.Flags().GetInt("max-events")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		var timeout time.Duration
		if timeoutStr != "" {
			var err error
			timeout, err = time.ParseDuration(timeoutStr)
			if err != nil {
				return fmt.Errorf("invalid timeout format: %w", err)
			}
		}

		tunnelCfg := &tunnelConfig{
			openclaw: &openclawConfig{
				enabled: openclawEnabled,
				url:     openclawURL,
				token:   openclawToken,
			},
			timeout:    timeout,
			maxEvents:  maxEvents,
			jsonOutput: jsonOutput,
		}

		// Convert HTTP URL to WebSocket URL
		wsURL := strings.Replace(serverURL, "http://", "ws://", 1)
		wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
		wsURL += "/tunnel?key=" + url.QueryEscape(cfg.APIKey)

		return runTunnelWithReconnect(wsURL, tunnelCfg)
	},
}

var tunnelStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check tunnel connection status",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL, _ := cmd.Flags().GetString("server")
		client, err := newClientWithServer(serverURL)
		if err != nil {
			return err
		}

		var status struct {
			Connected   bool `json:"connected"`
			Connections int  `json:"connections"`
		}

		if err := client.get("/api/v1/tunnel/status", &status); err != nil {
			return err
		}

		if status.Connected {
			fmt.Printf("Tunnel connected (%d active connection(s))\n", status.Connections)
		} else {
			fmt.Println("No tunnel connected")
		}

		return nil
	},
}

func runTunnelWithReconnect(wsURL string, cfg *tunnelConfig) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	backoff := initialBackoff
	reconnecting := false
	totalEvents := 0

	// Set up timeout if specified
	var timeoutCh <-chan time.Time
	var startTime time.Time
	if cfg.timeout > 0 {
		startTime = time.Now()
		timeoutCh = time.After(cfg.timeout)
	}

	for {
		if reconnecting {
			if !cfg.jsonOutput {
				fmt.Printf("Reconnecting in %v...\n", backoff)
			}
			select {
			case <-quit:
				if !cfg.jsonOutput {
					fmt.Println("\nShutting down...")
				}
				return nil
			case <-timeoutCh:
				outputResult(cfg, "timeout", nil, totalEvents, cfg.timeout)
				return nil
			case <-time.After(backoff):
			}
			// Exponential backoff
			backoff *= backoffFactor
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}

		conn, err := connectTunnel(wsURL)
		if err != nil {
			if !cfg.jsonOutput {
				fmt.Printf("Connection failed: %v\n", err)
			}
			reconnecting = true
			continue
		}

		// Reset backoff on successful connection
		backoff = initialBackoff
		reconnecting = false

		if !cfg.jsonOutput {
			fmt.Println("Tunnel connected! Waiting for events...")
			if cfg.openclaw.enabled {
				fmt.Printf("Forwarding to OpenClaw at %s\n", cfg.openclaw.url)
			}
			if cfg.timeout > 0 {
				fmt.Printf("Timeout: %v\n", cfg.timeout)
			}
			if cfg.maxEvents > 0 {
				fmt.Printf("Max events: %d\n", cfg.maxEvents)
			}
			fmt.Println("Press Ctrl+C to disconnect")
		}

		// Run tunnel until disconnect
		result := runTunnel(conn, quit, cfg, timeoutCh, &totalEvents)

		switch result.reason {
		case "quit":
			if !cfg.jsonOutput {
				fmt.Println("\nDisconnecting...")
			}
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			conn.Close()
			return nil
		case "timeout":
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			conn.Close()
			outputResult(cfg, "timeout", nil, totalEvents, time.Since(startTime))
			return nil
		case "max-events":
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			conn.Close()
			outputResult(cfg, "max-events", result.lastEvent, totalEvents, time.Since(startTime))
			return nil
		default:
			conn.Close()
			if !cfg.jsonOutput {
				fmt.Printf("Disconnected: %s\n", result.reason)
			}
			reconnecting = true
		}
	}
}

type tunnelResult struct {
	reason    string
	lastEvent map[string]interface{}
}

func outputResult(cfg *tunnelConfig, status string, lastEvent map[string]interface{}, eventCount int, elapsed time.Duration) {
	if cfg.jsonOutput {
		result := map[string]interface{}{
			"status":      status,
			"event_count": eventCount,
			"elapsed":     elapsed.String(),
		}
		if lastEvent != nil {
			result["last_event"] = lastEvent
		}
		jsonBytes, _ := json.Marshal(result)
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Printf("\nStopped: %s (received %d events in %v)\n", status, eventCount, elapsed.Round(time.Second))
	}
}

func connectTunnel(wsURL string) (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func runTunnel(conn *websocket.Conn, quit chan os.Signal, cfg *tunnelConfig, timeoutCh <-chan time.Time, totalEvents *int) tunnelResult {
	done := make(chan tunnelResult, 1)
	eventCh := make(chan map[string]interface{}, 10)

	// Read messages
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					done <- tunnelResult{reason: "server closed"}
				} else {
					done <- tunnelResult{reason: err.Error()}
				}
				return
			}

			var event map[string]interface{}
			if json.Unmarshal(message, &event) == nil {
				eventCh <- event
			}
		}
	}()

	// Ping to keep alive
	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case result := <-done:
				done <- result // Put it back
				return
			}
		}
	}()

	var lastEvent map[string]interface{}

	for {
		select {
		case <-quit:
			return tunnelResult{reason: "quit"}
		case <-timeoutCh:
			return tunnelResult{reason: "timeout", lastEvent: lastEvent}
		case result := <-done:
			return result
		case event := <-eventCh:
			lastEvent = event
			*totalEvents++

			if cfg.jsonOutput {
				// Output each event as JSON
				eventJSON, _ := json.Marshal(map[string]interface{}{
					"status": "event",
					"event":  event,
				})
				fmt.Println(string(eventJSON))
			} else {
				displayEvent(event)
			}

			// Forward to OpenClaw if enabled
			if cfg.openclaw.enabled {
				go forwardToOpenClaw(event, cfg.openclaw)
			}

			// Check max events
			if cfg.maxEvents > 0 && *totalEvents >= cfg.maxEvents {
				return tunnelResult{reason: "max-events", lastEvent: lastEvent}
			}
		}
	}
}

func displayEvent(event map[string]interface{}) {
	source, _ := event["source"].(string)
	eventType, _ := event["event"].(string)
	summary, _ := event["summary"].(string)
	timestamp, _ := event["timestamp"].(string)

	t, err := time.Parse(time.RFC3339, timestamp)
	if err == nil {
		timestamp = t.Local().Format("15:04:05")
	}

	fmt.Printf("\n[%s] %s/%s\n", timestamp, source, eventType)
	fmt.Printf("  %s\n", summary)
}

// forwardToOpenClaw sends the event to OpenClaw's /hooks/agent endpoint
func forwardToOpenClaw(event map[string]interface{}, ocConfig *openclawConfig) {
	source, _ := event["source"].(string)
	eventType, _ := event["event"].(string)
	summary, _ := event["summary"].(string)
	data := event["data"]

	// Build message for OpenClaw
	message := fmt.Sprintf("Webhook event received from %s:\n\nEvent: %s\nSummary: %s\n\nFull payload:\n```json\n%s\n```\n\nPlease process this event appropriately.",
		source, eventType, summary, prettyJSON(data))

	// Build OpenClaw payload
	payload := map[string]interface{}{
		"message":    message,
		"name":       fmt.Sprintf("openhook:%s", source),
		"sessionKey": fmt.Sprintf("hook:openhook:%s:%s", source, eventType),
		"wakeMode":   "now",
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("  [openclaw] Failed to marshal payload: %v\n", err)
		return
	}

	// Send to OpenClaw
	req, err := http.NewRequest("POST", ocConfig.url+"/hooks/agent", bytes.NewReader(jsonPayload))
	if err != nil {
		fmt.Printf("  [openclaw] Failed to create request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ocConfig.token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("  [openclaw] Failed to send: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("  [openclaw] Forwarded to OpenClaw\n")
	} else {
		fmt.Printf("  [openclaw] OpenClaw returned status %d\n", resp.StatusCode)
	}
}

func prettyJSON(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// listenCmd is an alias for tunnel start (for convenience)
var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Listen for webhook events (alias for 'tunnel start')",
	Long: `Listen for webhook events in real-time.

Use --openclaw to forward events to OpenClaw's /hooks/agent endpoint:
  openhook listen --openclaw --openclaw-token $OPENCLAW_HOOKS_TOKEN

This will send each webhook event to your local OpenClaw instance,
allowing your AI agent to react to GitHub pushes, Stripe payments, etc.`,
	RunE: tunnelStartCmd.RunE,
}

// Daemon commands
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the openhook background daemon",
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the webhook listener as a background daemon",
	Long: `Start openhook as a background daemon that runs continuously.

The daemon will automatically reconnect if disconnected and forward
events to OpenClaw if configured.

Examples:
  openhook daemon start
  openhook daemon start --openclaw`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if already running
		if pid := getDaemonPID(); pid > 0 {
			if isProcessRunning(pid) {
				return fmt.Errorf("daemon already running (PID %d)", pid)
			}
		}

		// Build command args for the daemon process
		daemonArgs := []string{"listen"}

		if openclawEnabled, _ := cmd.Flags().GetBool("openclaw"); openclawEnabled {
			daemonArgs = append(daemonArgs, "--openclaw")
		}
		if openclawURL, _ := cmd.Flags().GetString("openclaw-url"); openclawURL != "" {
			daemonArgs = append(daemonArgs, "--openclaw-url", openclawURL)
		}
		if openclawToken, _ := cmd.Flags().GetString("openclaw-token"); openclawToken != "" {
			daemonArgs = append(daemonArgs, "--openclaw-token", openclawToken)
		}
		if server, _ := cmd.Flags().GetString("server"); server != "" {
			daemonArgs = append(daemonArgs, "--server", server)
		}

		// Get the executable path
		executable, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}

		// Create log file
		logPath := getLogPath()
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to create log file: %w", err)
		}

		// Start the daemon process
		daemonCmd := exec.Command(executable, daemonArgs...)
		daemonCmd.Stdout = logFile
		daemonCmd.Stderr = logFile
		daemonCmd.Env = os.Environ()

		// Detach from parent
		daemonCmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}

		if err := daemonCmd.Start(); err != nil {
			logFile.Close()
			return fmt.Errorf("failed to start daemon: %w", err)
		}

		// Save PID
		if err := saveDaemonPID(daemonCmd.Process.Pid); err != nil {
			return fmt.Errorf("daemon started but failed to save PID: %w", err)
		}

		fmt.Printf("Daemon started (PID %d)\n", daemonCmd.Process.Pid)
		fmt.Printf("Logs: %s\n", logPath)
		return nil
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the background daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := getDaemonPID()
		if pid <= 0 {
			return fmt.Errorf("daemon not running (no PID file)")
		}

		if !isProcessRunning(pid) {
			removePIDFile()
			return fmt.Errorf("daemon not running (stale PID file removed)")
		}

		// Send SIGTERM
		process, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("failed to find process: %w", err)
		}

		if err := process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to stop daemon: %w", err)
		}

		// Wait a bit for graceful shutdown
		for i := 0; i < 10; i++ {
			time.Sleep(200 * time.Millisecond)
			if !isProcessRunning(pid) {
				break
			}
		}

		removePIDFile()
		fmt.Println("Daemon stopped")
		return nil
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if the daemon is running",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid := getDaemonPID()
		if pid <= 0 {
			fmt.Println("Daemon is not running")
			return nil
		}

		if !isProcessRunning(pid) {
			fmt.Println("Daemon is not running (stale PID file)")
			return nil
		}

		fmt.Printf("Daemon is running (PID %d)\n", pid)
		fmt.Printf("Logs: %s\n", getLogPath())
		return nil
	},
}

var daemonLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show daemon logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		follow, _ := cmd.Flags().GetBool("follow")
		lines, _ := cmd.Flags().GetInt("lines")

		logPath := getLogPath()
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			return fmt.Errorf("no log file found at %s", logPath)
		}

		if follow {
			// Use tail -f
			tailCmd := exec.Command("tail", "-f", "-n", strconv.Itoa(lines), logPath)
			tailCmd.Stdout = os.Stdout
			tailCmd.Stderr = os.Stderr
			return tailCmd.Run()
		}

		// Just show last N lines
		tailCmd := exec.Command("tail", "-n", strconv.Itoa(lines), logPath)
		tailCmd.Stdout = os.Stdout
		tailCmd.Stderr = os.Stderr
		return tailCmd.Run()
	},
}

// Helper functions for daemon management
func getConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openhook")
}

func getPIDPath() string {
	return filepath.Join(getConfigDir(), "daemon.pid")
}

func getLogPath() string {
	return filepath.Join(getConfigDir(), "daemon.log")
}

func getDaemonPID() int {
	data, err := os.ReadFile(getPIDPath())
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}

func saveDaemonPID(pid int) error {
	return os.WriteFile(getPIDPath(), []byte(strconv.Itoa(pid)), 0644)
}

func removePIDFile() {
	os.Remove(getPIDPath())
}

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds, so we need to send signal 0
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func init() {
	// Tunnel start flags
	tunnelStartCmd.Flags().String("server", "", "Server URL")
	tunnelStartCmd.Flags().Bool("openclaw", false, "Forward events to OpenClaw's /hooks/agent endpoint")
	tunnelStartCmd.Flags().String("openclaw-url", "http://127.0.0.1:18789", "OpenClaw Gateway URL")
	tunnelStartCmd.Flags().String("openclaw-token", "", "OpenClaw hooks token (or set OPENCLAW_HOOKS_TOKEN)")
	tunnelStartCmd.Flags().String("timeout", "", "Auto-stop after duration (e.g., 30s, 5m, 1h)")
	tunnelStartCmd.Flags().Int("max-events", 0, "Auto-stop after receiving N events")
	tunnelStartCmd.Flags().Bool("json", false, "Output events as JSON (for programmatic use)")

	// Listen command (alias) - same flags
	listenCmd.Flags().String("server", "", "Server URL")
	listenCmd.Flags().Bool("openclaw", false, "Forward events to OpenClaw's /hooks/agent endpoint")
	listenCmd.Flags().String("openclaw-url", "http://127.0.0.1:18789", "OpenClaw Gateway URL")
	listenCmd.Flags().String("openclaw-token", "", "OpenClaw hooks token (or set OPENCLAW_HOOKS_TOKEN)")
	listenCmd.Flags().String("timeout", "", "Auto-stop after duration (e.g., 30s, 5m, 1h)")
	listenCmd.Flags().Int("max-events", 0, "Auto-stop after receiving N events")
	listenCmd.Flags().Bool("json", false, "Output events as JSON (for programmatic use)")

	tunnelStatusCmd.Flags().String("server", "", "Server URL")

	// Daemon start flags
	daemonStartCmd.Flags().String("server", "", "Server URL")
	daemonStartCmd.Flags().Bool("openclaw", false, "Forward events to OpenClaw's /hooks/agent endpoint")
	daemonStartCmd.Flags().String("openclaw-url", "http://127.0.0.1:18789", "OpenClaw Gateway URL")
	daemonStartCmd.Flags().String("openclaw-token", "", "OpenClaw hooks token (or set OPENCLAW_HOOKS_TOKEN)")

	// Daemon logs flags
	daemonLogsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
	daemonLogsCmd.Flags().IntP("lines", "n", 50, "Number of lines to show")

	tunnelCmd.AddCommand(tunnelStartCmd)
	tunnelCmd.AddCommand(tunnelStatusCmd)
	rootCmd.AddCommand(tunnelCmd)
	rootCmd.AddCommand(listenCmd)

	// Daemon commands
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonLogsCmd)
	rootCmd.AddCommand(daemonCmd)
}
