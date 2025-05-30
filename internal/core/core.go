package core

import (
	"github.com/denchenko/gg/internal/config"
	"github.com/denchenko/gg/internal/core/app"
	do "github.com/samber/do/v2"
)

var Package = do.Package(
	do.Lazy[*app.App](NewApp),
)

// NewApp creates a new App instance with dependencies from the injector.
func NewApp(i do.Injector) (*app.App, error) {
	cfg := do.MustInvoke[*config.Config](i)
	repo := do.MustInvoke[app.Repository](i)
	return app.NewApp(cfg, repo)
}
