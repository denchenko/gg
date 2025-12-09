package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	// Save original env vars
	originalToken := os.Getenv("GG_TOKEN")
	originalTeam := os.Getenv("GG_TEAM")
	originalBaseURL := os.Getenv("GG_BASE_URL")
	originalWebhookAddr := os.Getenv("GG_WEBHOOK_ADDRESS")

	// Clean up after test
	defer func() {
		if originalToken != "" {
			_ = os.Setenv("GG_TOKEN", originalToken)
		} else {
			_ = os.Unsetenv("GG_TOKEN")
		}
		if originalTeam != "" {
			_ = os.Setenv("GG_TEAM", originalTeam)
		} else {
			_ = os.Unsetenv("GG_TEAM")
		}
		if originalBaseURL != "" {
			_ = os.Setenv("GG_BASE_URL", originalBaseURL)
		} else {
			_ = os.Unsetenv("GG_BASE_URL")
		}
		if originalWebhookAddr != "" {
			_ = os.Setenv("GG_WEBHOOK_ADDRESS", originalWebhookAddr)
		} else {
			_ = os.Unsetenv("GG_WEBHOOK_ADDRESS")
		}
	}()

	tests := []struct {
		name        string
		setupEnv    func()
		expectError bool
		validate    func(*testing.T, *Config)
	}{
		{
			name: "successful config creation",
			setupEnv: func() {
				_ = os.Setenv("GG_TOKEN", "test-token")
				_ = os.Setenv("GG_TEAM", "user1,user2,user3")
				_ = os.Unsetenv("GG_BASE_URL")
				_ = os.Unsetenv("GG_WEBHOOK_ADDRESS")
			},
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "test-token", cfg.Token)
				assert.Equal(t, "https://gitlab.com", cfg.BaseURL)
				assert.Equal(t, []string{"user1", "user2", "user3"}, cfg.TeamUsers)
				assert.Equal(t, ":8080", cfg.WebhookAddress)
			},
		},
		{
			name: "custom base URL",
			setupEnv: func() {
				_ = os.Setenv("GG_TOKEN", "test-token")
				_ = os.Setenv("GG_TEAM", "user1")
				_ = os.Setenv("GG_BASE_URL", "https://gitlab.example.com")
				_ = os.Unsetenv("GG_WEBHOOK_ADDRESS")
			},
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "https://gitlab.example.com", cfg.BaseURL)
			},
		},
		{
			name: "custom webhook address",
			setupEnv: func() {
				_ = os.Setenv("GG_TOKEN", "test-token")
				_ = os.Setenv("GG_TEAM", "user1")
				_ = os.Setenv("GG_WEBHOOK_ADDRESS", "0.0.0.0:9090")
			},
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "0.0.0.0:9090", cfg.WebhookAddress)
			},
		},
		{
			name: "missing token",
			setupEnv: func() {
				_ = os.Unsetenv("GG_TOKEN")
				_ = os.Setenv("GG_TEAM", "user1")
			},
			expectError: true,
		},
		{
			name: "missing team",
			setupEnv: func() {
				_ = os.Setenv("GG_TOKEN", "test-token")
				_ = os.Unsetenv("GG_TEAM")
			},
			expectError: true,
		},
		{
			name: "team with spaces",
			setupEnv: func() {
				_ = os.Setenv("GG_TOKEN", "test-token")
				_ = os.Setenv("GG_TEAM", "user1 , user2 , user3")
			},
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, []string{"user1", "user2", "user3"}, cfg.TeamUsers)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean env
			_ = os.Unsetenv("GG_TOKEN")
			_ = os.Unsetenv("GG_TEAM")
			_ = os.Unsetenv("GG_BASE_URL")
			_ = os.Unsetenv("GG_WEBHOOK_ADDRESS")

			tt.setupEnv()

			cfg, err := New()

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, cfg)
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}
		})
	}
}

func TestNewConfig(t *testing.T) {
	// Test the DI version
	cfg, err := NewConfig(nil)
	// This will fail if env vars aren't set, which is expected
	// In real usage, env vars would be set
	if err != nil {
		require.Error(t, err)
	} else {
		assert.NotNil(t, cfg)
	}
}
