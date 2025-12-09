package config

import (
	"errors"
	"os"
	"strings"

	do "github.com/samber/do/v2"
)

var Package = do.Package(
	do.Lazy[*Config](NewConfig),
)

// Config holds the application configuration.
type Config struct {
	BaseURL        string
	Token          string
	TeamUsers      []string
	WebhookAddress string
}

// NewConfig creates a new configuration from environment variables (for DI).
func NewConfig(_ do.Injector) (*Config, error) {
	return New()
}

// New creates a new configuration from environment variables.
func New() (*Config, error) {
	gitServiceURL := os.Getenv("GG_BASE_URL")
	if gitServiceURL == "" {
		gitServiceURL = "https://gitlab.com"
	}

	webhookAddress := os.Getenv("GG_WEBHOOK_ADDRESS")
	if webhookAddress == "" {
		webhookAddress = ":8080"
	}

	privateToken := os.Getenv("GG_TOKEN")
	if privateToken == "" {
		return nil, errors.New("GG_TOKEN environment variable is required")
	}

	teamUsersStr := os.Getenv("GG_TEAM")
	if teamUsersStr == "" {
		return nil, errors.New("GG_TEAM environment variable is required")
	}

	teamUsers := strings.Split(teamUsersStr, ",")
	for i, user := range teamUsers {
		teamUsers[i] = strings.TrimSpace(user)
	}

	return &Config{
		BaseURL:        gitServiceURL,
		Token:          privateToken,
		TeamUsers:      teamUsers,
		WebhookAddress: webhookAddress,
	}, nil
}
