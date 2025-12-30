package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/denchenko/gg/internal/core/app"
	"github.com/denchenko/gg/internal/core/domain"
	"github.com/denchenko/gg/internal/issue"
	"github.com/denchenko/gg/internal/log"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
)

func Issue(appInstance *app.App, issuer *issue.Issuer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Issues",
	}

	cmd.AddCommand(newIssueBrowseCommand(appInstance, issuer))

	return cmd
}

func newIssueBrowseCommand(appInstance *app.App, issuer *issue.Issuer) *cobra.Command {
	return &cobra.Command{
		Use:   "browse",
		Short: "Open issue in browser",
		Long:  `Open the issue linked to the merge request for the current git branch in your default browser.`,
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return browseIssue(appInstance, issuer)
		},
	}
}

func browseIssue(appInstance *app.App, issuer *issue.Issuer) error {
	ctx := context.Background()

	// Get current project and branch
	var (
		err            error
		currentProject *domain.Project
		currentBranch  string
	)

	err = log.WithSpinner("Getting current project info...", func() error {
		var err error
		currentProject, currentBranch, err = appInstance.GetCurrentProjectInfo(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current project info: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to get current project info: %w", err)
	}

	// Get MR for current branch
	var mr *domain.MergeRequest
	err = log.WithSpinner("Finding merge request...", func() error {
		var err error
		mr, err = appInstance.GetMergeRequestByBranch(ctx, currentProject.ID, currentBranch)
		if err != nil {
			return fmt.Errorf("failed to find merge request for branch %s: %w", currentBranch, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to find merge request: %w", err)
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

	// Open URL in browser
	if err := open.Start(issueURL); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	return nil
}
