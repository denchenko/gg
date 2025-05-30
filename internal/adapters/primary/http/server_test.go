package http

import (
	"context"
	"testing"
	"time"

	"github.com/denchenko/gg/internal/core/app"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	appInstance := &app.App{}

	server := NewServer(":8080", appInstance)

	assert.NotNil(t, server)
	assert.NotNil(t, server.server)
	assert.Equal(t, ":8080", server.server.Addr)
	assert.Equal(t, appInstance, server.app)
}

func TestServer_Start(t *testing.T) {
	// This test would require actually starting a server, which is complex
	// In a real scenario, you'd test this with integration tests
	// For now, we'll just verify the server can be created
	appInstance := &app.App{}

	server := NewServer(":0", appInstance) // Use :0 to get a free port
	assert.NotNil(t, server)
}

func TestServer_Shutdown(t *testing.T) {
	appInstance := &app.App{}

	server := NewServer(":0", appInstance)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Server not started, so shutdown should work
	err := server.Shutdown(ctx)
	assert.NoError(t, err)
}
