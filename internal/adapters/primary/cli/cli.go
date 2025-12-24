package cli

import (
	"github.com/denchenko/gg/internal/adapters/primary/cli/commands"
	"github.com/denchenko/gg/internal/config"
	"github.com/denchenko/gg/internal/core/app"
	do "github.com/samber/do/v2"
	"github.com/spf13/cobra"
)

// Command creates and returns the root CLI command.
func Command(i do.Injector) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Long: `A CLI tool for managing GitLab.`,
	}

	appInstance := do.MustInvoke[*app.App](i)
	cfg := do.MustInvoke[*config.Config](i)

	cmd.AddCommand(
		commands.My(cfg, appInstance),
		commands.Team(appInstance),
		commands.MR(cfg, appInstance),
	)

	return cmd, nil
}
