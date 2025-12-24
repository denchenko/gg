package adapters

import (
	"fmt"

	"github.com/denchenko/gg/internal/adapters/primary/cli"
	httpadapter "github.com/denchenko/gg/internal/adapters/primary/http"
	"github.com/denchenko/gg/internal/adapters/secondary/cache"
	"github.com/denchenko/gg/internal/adapters/secondary/repository/cached"
	"github.com/denchenko/gg/internal/adapters/secondary/repository/gitlab"
	"github.com/denchenko/gg/internal/config"
	"github.com/denchenko/gg/internal/core/app"
	do "github.com/samber/do/v2"
	"github.com/spf13/cobra"
	glclient "gitlab.com/gitlab-org/api/client-go"
)

var PrimaryPackage = do.Package(
	do.Lazy[*cobra.Command](cli.Command),
	do.Lazy[*httpadapter.Server](NewHTTPServer),
)

var SecondaryPackage = do.Package(
	do.Lazy[*glclient.Client](NewGitLabClient),
	do.Lazy[*gitlab.Repository](NewGitLabRepository),
	do.Lazy[cache.Cache](NewCache),
	do.Lazy[app.Repository](NewRepository),
)

// NewGitLabClient creates a new GitLab client.
func NewGitLabClient(i do.Injector) (*glclient.Client, error) {
	cfg := do.MustInvoke[*config.Config](i)
	client, err := glclient.NewClient(cfg.Token, glclient.WithBaseURL(cfg.BaseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	return client, nil
}

// NewGitLabRepository creates a new GitLab repository instance.
func NewGitLabRepository(i do.Injector) (*gitlab.Repository, error) {
	client := do.MustInvoke[*glclient.Client](i)
	cfg := do.MustInvoke[*config.Config](i)

	return gitlab.NewRepository(client, cfg.BaseURL), nil
}

// NewCache creates a new cache instance.
func NewCache(_ do.Injector) (cache.Cache, error) {
	return cache.NewInMemoryCache(), nil
}

// NewRepository creates a repository adapter that implements app.Repository.
// It wraps the GitLab repository with a cached repository for performance.
func NewRepository(i do.Injector) (app.Repository, error) {
	gitlabRepo := do.MustInvoke[*gitlab.Repository](i)
	cacheInstance := do.MustInvoke[cache.Cache](i)

	return cached.NewCachedRepository(gitlabRepo, cacheInstance), nil
}

// NewHTTPServer creates a new HTTP server.
func NewHTTPServer(i do.Injector) (*httpadapter.Server, error) {
	appInstance := do.MustInvoke[*app.App](i)
	cfg := do.MustInvoke[*config.Config](i)

	return httpadapter.NewServer(cfg.WebhookAddress, appInstance), nil
}
