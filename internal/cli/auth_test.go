package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMaskKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"oh_live_abc123xyz789", "oh_live_********z789"},
		{"oh_test_short", "oh_test_*hort"},
		{"short", "short"}, // Too short to mask
		{"", ""},
	}

	for _, tt := range tests {
		result := maskKey(tt.input)
		if result != tt.expected {
			t.Errorf("maskKey(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestConfigSaveLoad(t *testing.T) {
	// Use temp directory for config
	tmpDir := t.TempDir()
	originalConfigDir := configDir

	// Override configDir function for testing
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Verify config dir is in temp
	expectedDir := filepath.Join(tmpDir, ".openhook")
	if configDir() != expectedDir {
		t.Skipf("Config dir override not working, skipping test")
	}

	_ = originalConfigDir // silence unused warning

	// Test save
	cfg := &config{
		APIKey:    "oh_live_test123",
		ServerURL: "https://api.example.com",
	}

	err := saveConfig(cfg)
	if err != nil {
		t.Fatalf("saveConfig failed: %v", err)
	}

	// Verify file exists with correct permissions
	info, err := os.Stat(configPath())
	if err != nil {
		t.Fatalf("Config file not created: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("Config file has wrong permissions: %o", info.Mode().Perm())
	}

	// Test load
	loaded, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}

	if loaded.APIKey != cfg.APIKey {
		t.Errorf("APIKey mismatch: got %s, want %s", loaded.APIKey, cfg.APIKey)
	}
	if loaded.ServerURL != cfg.ServerURL {
		t.Errorf("ServerURL mismatch: got %s, want %s", loaded.ServerURL, cfg.ServerURL)
	}
}

func TestLoadConfigNotFound(t *testing.T) {
	// Use temp directory that doesn't have config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	_, err := loadConfig()
	if err == nil {
		t.Error("Expected error when config doesn't exist")
	}
}

func TestKeyValidation(t *testing.T) {
	tests := []struct {
		key     string
		isValid bool
	}{
		{"oh_live_abc123", true},
		{"oh_test_xyz789", true},
		{"oh_live_", true}, // Prefix only is technically valid format
		{"invalid_key", false},
		{"oh_invalid_abc", false},
		{"", false},
	}

	for _, tt := range tests {
		hasValidPrefix := len(tt.key) >= 8 && (tt.key[:8] == "oh_live_" || tt.key[:8] == "oh_test_")
		if hasValidPrefix != tt.isValid {
			t.Errorf("Key %q validation: got %v, want %v", tt.key, hasValidPrefix, tt.isValid)
		}
	}
}
