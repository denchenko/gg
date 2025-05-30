package commands

import (
	"context"
	"fmt"

	"github.com/denchenko/gg/internal/core/app"
	"github.com/denchenko/gg/internal/core/domain"
	"github.com/denchenko/gg/internal/format/ascii"
	"github.com/denchenko/gg/internal/log"
	"github.com/spf13/cobra"
)

func My(appInstance *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "my",
		Short: "Everything related to you",
	}

	cmd.AddCommand(
		MyMR(appInstance),
		MyReview(appInstance),
	)

	return cmd
}

func MyMR(appInstance *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "mr",
		Short: "Show your merge requests status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showMyMRStatus(appInstance)
		},
	}
}

func MyReview(appInstance *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "review",
		Short: "Show your review workload",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showMyReviewWorkload(appInstance)
		},
	}
}

func showMyReviewWorkload(appInstance *app.App) error {
	ctx := context.Background()

	var mrsWithStatus []*domain.MergeRequestWithStatus
	err := log.WithSpinner("Fetching your review workload...", func() error {
		mergeRequestsWithStatus, err := appInstance.GetMyReviewWorkloadWithStatus(ctx)
		if err != nil {
			return err
		}

		mrsWithStatus = mergeRequestsWithStatus

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to get review workload: %w", err)
	}

	formatted, err := ascii.FormatMyReviewWorkload(mrsWithStatus)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	fmt.Print(formatted)

	return nil
}

func showMyMRStatus(appInstance *app.App) error {
	ctx := context.Background()

	var mrsWithStatus []*domain.MergeRequestWithStatus
	err := log.WithSpinner("Fetching your merge requests...", func() error {
		mergeRequestsWithStatus, err := appInstance.GetMergeRequestsWithStatus(ctx)
		if err != nil {
			return err
		}

		mrsWithStatus = mergeRequestsWithStatus
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to get merge requests: %w", err)
	}

	formatted, err := ascii.FormatMyMergeRequestStatus(mrsWithStatus)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	fmt.Print(formatted)

	return nil
}
