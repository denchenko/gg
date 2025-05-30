package commands

import (
	"testing"

	"github.com/denchenko/gg/internal/config"
	"github.com/denchenko/gg/internal/core/app"
	"github.com/stretchr/testify/assert"
)

func TestParseMRURL(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		mrURL       string
		expectError bool
		expected    struct {
			projectPath string
			mrID        int
		}
	}{
		{
			name:        "valid HTTPS URL",
			baseURL:     "https://gitlab.com",
			mrURL:       "https://gitlab.com/group/project/-/merge_requests/123",
			expectError: false,
			expected: struct {
				projectPath string
				mrID        int
			}{
				projectPath: "group/project",
				mrID:        123,
			},
		},
		{
			name:        "valid URL with custom base",
			baseURL:     "https://gitlab.example.com",
			mrURL:       "https://gitlab.example.com/org/repo/-/merge_requests/456",
			expectError: false,
			expected: struct {
				projectPath string
				mrID        int
			}{
				projectPath: "org/repo",
				mrID:        456,
			},
		},
		{
			name:        "invalid URL format",
			baseURL:     "https://gitlab.com",
			mrURL:       "https://gitlab.com/group/project",
			expectError: true,
		},
		{
			name:        "invalid MR ID",
			baseURL:     "https://gitlab.com",
			mrURL:       "https://gitlab.com/group/project/-/merge_requests/abc",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectPath, mrID, err := parseMRURL(tt.baseURL, tt.mrURL)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.projectPath, projectPath)
				assert.Equal(t, tt.expected.mrID, mrID)
			}
		})
	}
}

func TestMR(t *testing.T) {
	cfg := &config.Config{BaseURL: "https://gitlab.com"}
	appInstance := &app.App{}

	cmd := MR(cfg, appInstance)

	assert.NotNil(t, cmd)
	assert.Equal(t, "mr", cmd.Use)
}
