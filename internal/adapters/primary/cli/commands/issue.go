package commands

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/denchenko/gg/internal/core/app"
	"github.com/denchenko/gg/internal/issue"
	"github.com/spf13/cobra"
)

func Issue(appInstance *app.App, issuer *issue.Issuer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Issues",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "browse",
		Short: "Open issue in browser",
		Long:  `Open the issue linked to the merge request for the current git branch in your default browser.`,
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx := context.Background()

			// Get current project and branch
			currentProject, currentBranch, err := appInstance.GetCurrentProjectInfo(ctx)
			if err != nil {
				return fmt.Errorf("failed to get current project info: %w", err)
			}

			// Get MR for current branch
			mr, err := appInstance.GetMergeRequestByBranch(ctx, currentProject.ID, currentBranch)
			if err != nil {
				return fmt.Errorf("failed to find merge request for branch %s: %w", currentBranch, err)
			}

			// Extract issue number from MR title
			issueNumber := issuer.ExtractNumber(mr.Title)
			if issueNumber == "" {
				return fmt.Errorf("no issue number found in merge request title: %s", mr.Title)
			}

			// Generate issue URL
			issueURL, err := issuer.MakeURL(issueNumber)
			if err != nil {
				return fmt.Errorf("failed to generate issue URL: %w", err)
			}

			if issueURL == "" {
				return errors.New("issue URL template is not configured (GG_ISSUE_URL_TEMPLATE)")
			}

			// Open URL in browser using open
			cmd := exec.CommandContext(ctx, "open", issueURL)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to open browser: %w", err)
			}

			return nil
		},
	})

	return cmd
}
