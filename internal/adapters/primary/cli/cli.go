package cli

import (
	"github.com/denchenko/gg/internal/adapters/primary/cli/commands"
	"github.com/denchenko/gg/internal/config"
	"github.com/denchenko/gg/internal/core/app"
	ascii "github.com/denchenko/gg/internal/format/ascii"
	"github.com/denchenko/gg/internal/issue"
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
	issuer := do.MustInvoke[*issue.Issuer](i)
	formatter := do.MustInvoke[*ascii.Formatter](i)

	cmd.AddCommand(
		commands.My(cfg, appInstance, formatter),
		commands.Team(cfg, appInstance, formatter),
		commands.MR(cfg, appInstance, formatter),
		commands.Issue(appInstance, issuer),
	)

	return cmd, nil
}
