package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIClient_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Missing or wrong auth header")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := &apiClient{serverURL: server.URL, apiKey: "test-key"}

	var result map[string]string
	err := client.get("/test", &result)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("Expected status ok, got %s", result["status"])
	}
}

func TestAPIClient_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Missing content-type header")
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "test" {
			t.Errorf("Expected name=test, got %s", body["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "123"})
	}))
	defer server.Close()

	client := &apiClient{serverURL: server.URL, apiKey: "test-key"}

	var result map[string]string
	err := client.post("/test", map[string]string{"name": "test"}, &result)
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}

	if result["id"] != "123" {
		t.Errorf("Expected id 123, got %s", result["id"])
	}
}

func TestAPIClient_Delete(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := &apiClient{serverURL: server.URL, apiKey: "test-key"}

	err := client.delete("/test/123")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if !called {
		t.Error("Server was not called")
	}
}

func TestAPIClient_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
	}))
	defer server.Close()

	client := &apiClient{serverURL: server.URL, apiKey: "test-key"}

	var result map[string]string
	err := client.get("/test", &result)
	if err == nil {
		t.Fatal("Expected error")
	}

	if err.Error() != "invalid request" {
		t.Errorf("Expected 'invalid request', got '%s'", err.Error())
	}
}

func TestAPIClient_CreateSubscription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/subscriptions" {
			t.Errorf("Wrong path: %s", r.URL.Path)
		}

		var req struct {
			Platform string   `json:"platform"`
			Target   string   `json:"target"`
			Events   []string `json:"events"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		if req.Platform != "github" || req.Target != "owner/repo" {
			t.Errorf("Wrong request body: %+v", req)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Subscription{
			ID:       "sub_123",
			Platform: req.Platform,
			Target:   req.Target,
			Events:   req.Events,
			Status:   "active",
		})
	}))
	defer server.Close()

	client := &apiClient{serverURL: server.URL, apiKey: "test-key"}

	sub, err := client.CreateSubscription("github", "owner/repo", []string{"push"})
	if err != nil {
		t.Fatalf("CreateSubscription failed: %v", err)
	}

	if sub.ID != "sub_123" {
		t.Errorf("Expected id sub_123, got %s", sub.ID)
	}
	if sub.Platform != "github" {
		t.Errorf("Expected platform github, got %s", sub.Platform)
	}
}

func TestAPIClient_ListSubscriptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		platform := r.URL.Query().Get("platform")

		subs := []Subscription{
			{ID: "sub_1", Platform: "github", Target: "owner/repo1"},
			{ID: "sub_2", Platform: "stripe", Target: ""},
		}

		if platform != "" {
			filtered := []Subscription{}
			for _, s := range subs {
				if s.Platform == platform {
					filtered = append(filtered, s)
				}
			}
			subs = filtered
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(subs)
	}))
	defer server.Close()

	client := &apiClient{serverURL: server.URL, apiKey: "test-key"}

	// List all
	subs, err := client.ListSubscriptions("")
	if err != nil {
		t.Fatalf("ListSubscriptions failed: %v", err)
	}
	if len(subs) != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", len(subs))
	}

	// Filter by platform
	subs, err = client.ListSubscriptions("github")
	if err != nil {
		t.Fatalf("ListSubscriptions with filter failed: %v", err)
	}
	if len(subs) != 1 {
		t.Errorf("Expected 1 subscription, got %d", len(subs))
	}
}

func TestAPIClient_GetMe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/me" {
			t.Errorf("Wrong path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Me{
			ID:                 "user_123",
			Email:              "test@example.com",
			ConnectedPlatforms: []string{"github", "stripe"},
		})
	}))
	defer server.Close()

	client := &apiClient{serverURL: server.URL, apiKey: "test-key"}

	me, err := client.GetMe()
	if err != nil {
		t.Fatalf("GetMe failed: %v", err)
	}

	if me.Email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", me.Email)
	}
	if len(me.ConnectedPlatforms) != 2 {
		t.Errorf("Expected 2 connected platforms, got %d", len(me.ConnectedPlatforms))
	}
}
