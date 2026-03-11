package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const defaultServerURL = "https://api.openhook.dev"

type apiClient struct {
	serverURL string
	apiKey    string
}

func newClient() (*apiClient, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("not authenticated, run: openhook auth login --key <your-key>")
	}

	serverURL := cfg.ServerURL
	if serverURL == "" {
		serverURL = defaultServerURL
	}

	return &apiClient{
		serverURL: serverURL,
		apiKey:    cfg.APIKey,
	}, nil
}

func newClientWithServer(serverOverride string) (*apiClient, error) {
	client, err := newClient()
	if err != nil {
		return nil, err
	}
	if serverOverride != "" {
		client.serverURL = serverOverride
	}
	return client, nil
}

func (c *apiClient) do(method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.serverURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("connect to server: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return fmt.Errorf("%s", errResp.Error)
		}
		return fmt.Errorf("request failed (status %d)", resp.StatusCode)
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("parse response: %w", err)
		}
	}

	return nil
}

func (c *apiClient) get(path string, result interface{}) error {
	return c.do(http.MethodGet, path, nil, result)
}

func (c *apiClient) post(path string, body, result interface{}) error {
	return c.do(http.MethodPost, path, body, result)
}

func (c *apiClient) delete(path string) error {
	return c.do(http.MethodDelete, path, nil, nil)
}

// Subscription represents a webhook subscription
type Subscription struct {
	ID        string   `json:"id"`
	Platform  string   `json:"platform"`
	Target    string   `json:"target"`
	Events    []string `json:"events"`
	Status    string   `json:"status"`
	CreatedAt string   `json:"created_at"`
}

func (c *apiClient) CreateSubscription(platform, target string, events []string) (*Subscription, error) {
	req := struct {
		Platform string   `json:"platform"`
		Target   string   `json:"target"`
		Events   []string `json:"events"`
	}{platform, target, events}

	var sub Subscription
	if err := c.post("/api/v1/subscriptions", req, &sub); err != nil {
		return nil, err
	}
	return &sub, nil
}

func (c *apiClient) ListSubscriptions(platform string) ([]Subscription, error) {
	path := "/api/v1/subscriptions"
	if platform != "" {
		path += "?platform=" + platform
	}

	var subs []Subscription
	if err := c.get(path, &subs); err != nil {
		return nil, err
	}
	return subs, nil
}

func (c *apiClient) DeleteSubscription(id string) error {
	return c.delete("/api/v1/subscriptions/" + id)
}

// Me represents the current user info
type Me struct {
	ID                 string   `json:"id"`
	Email              string   `json:"email"`
	ConnectedPlatforms []string `json:"connected_platforms"`
}

func (c *apiClient) GetMe() (*Me, error) {
	var me Me
	if err := c.get("/api/v1/auth/me", &me); err != nil {
		return nil, err
	}
	return &me, nil
}
