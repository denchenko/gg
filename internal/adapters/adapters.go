package adapters

import (
	"fmt"

	"github.com/denchenko/gg/internal/adapters/primary/cli"
	httpadapter "github.com/denchenko/gg/internal/adapters/primary/http"
	"github.com/denchenko/gg/internal/adapters/secondary/gitlab"
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

	return gitlab.NewRepository(client), nil
}

// NewRepository creates a repository adapter that implements app.Repository.
func NewRepository(i do.Injector) (app.Repository, error) {
	return do.MustInvoke[*gitlab.Repository](i), nil
}

// NewHTTPServer creates a new HTTP server.
func NewHTTPServer(i do.Injector) (*httpadapter.Server, error) {
	appInstance := do.MustInvoke[*app.App](i)
	addr := ":8080"

	return httpadapter.NewServer(addr, appInstance), nil
}
