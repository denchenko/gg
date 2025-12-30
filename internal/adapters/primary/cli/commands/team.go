package commands

import (
	"context"
	"fmt"

	"github.com/denchenko/gg/internal/config"
	"github.com/denchenko/gg/internal/core/app"
	"github.com/denchenko/gg/internal/core/domain"
	"github.com/denchenko/gg/internal/format/ascii"
	"github.com/denchenko/gg/internal/log"
	"github.com/spf13/cobra"
)

func Team(cfg *config.Config, appInstance *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "team",
		Short: "Everything related to your team",
	}

	cmd.AddCommand(
		TeamReview(cfg, appInstance),
	)

	return cmd
}

func TeamReview(cfg *config.Config, appInstance *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "review",
		Short: "Show your team workload",
		RunE: func(_ *cobra.Command, _ []string) error {
			return showTeamReviewWorkload(cfg, appInstance)
		},
	}
}

func showTeamReviewWorkload(cfg *config.Config, appInstance *app.App) error {
	ctx := context.Background()

	var workloads []*domain.UserWorkload
	err := log.WithSpinner("Analyzing team workload...", func() error {
		var err error
		workloads, err = appInstance.AnalyzeActiveMRs(ctx)
		if err != nil {
			return fmt.Errorf("failed to analyze workload: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to analyze workload: %w", err)
	}

	formattedOutput, err := ascii.FormatTeamWorkload(workloads, cfg.IssueURLTemplate)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	fmt.Println(formattedOutput)

	return nil
}
