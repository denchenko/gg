package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/denchenko/gg/internal/config"
	"github.com/denchenko/gg/internal/core/app"
	"github.com/denchenko/gg/internal/core/domain"
	ascii "github.com/denchenko/gg/internal/format/ascii"
	"github.com/denchenko/gg/internal/log"
	"github.com/spf13/cobra"
)

func MR(cfg *config.Config, appInstance *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mr",
		Short: "Merge Requests",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "roulette [MR_URL]",
		Short: "Suggest assignee and reviewer for a merge request",
		Long: `Analyze team review workload and suggest appropriate assignee and reviewer for a merge request.
If MR_URL is not provided, it will try to get the current merge request URL from git.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var mrURL string
			if len(args) > 0 {
				mrURL = args[0]
			} else {
				var err error
				mrURL, err = appInstance.GetCurrentMRURL(context.Background())
				if err != nil {
					return fmt.Errorf("failed to get current MR URL: %w", err)
				}
			}
			return suggestAssignees(cfg, appInstance, mrURL)
		},
	})
	return cmd
}

func suggestAssignees(cfg *config.Config, appInstance *app.App, mrURL string) error {
	ctx := context.Background()

	// TODO: common parse
	projectPath, mrID, err := parseMRURL(cfg.BaseURL, mrURL)
	if err != nil {
		return fmt.Errorf("failed to parse merge request URL: %w", err)
	}

	var project *domain.Project
	err = log.WithSpinner("Fetching project information...", func() error {
		var err error
		project, err = appInstance.GetProject(ctx, projectPath)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	var mr *domain.MergeRequest
	err = log.WithSpinner("Fetching merge request details...", func() error {
		var err error
		mr, err = appInstance.GetMergeRequest(ctx, project.ID, mrID)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to get merge request: %w", err)
	}

	var workloads []*domain.UserWorkload
	err = log.WithSpinner("Analyzing team workload...", func() error {
		var err error
		workloads, err = appInstance.AnalyzeWorkload(ctx, project.ID)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to analyze workload: %w", err)
	}

	var suggestedAssignee, suggestedReviewer *domain.User
	err = log.WithSpinner("Calculating suggestions...", func() error {
		var err error
		suggestedAssignee, suggestedReviewer, err = appInstance.SuggestAssigneeAndReviewer(ctx, mr, workloads)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to get suggestions: %w", err)
	}

	formatted, err := ascii.FormatMRRoulette(mr, mrURL, workloads, suggestedAssignee, suggestedReviewer)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	fmt.Print(formatted)

	if suggestedAssignee != nil || suggestedReviewer != nil {
		return applySuggestions(ctx, appInstance, project.ID, mrID, suggestedAssignee, suggestedReviewer)
	}

	return nil
}

func applySuggestions(ctx context.Context, appInstance *app.App, projectID, mrID int, suggestedAssignee, suggestedReviewer *domain.User) error {
	fmt.Printf("\nWould you like to apply these suggestions to the merge request? [y/N]: ")
	var response string
	if _, err := fmt.Scanln(&response); err != nil && err.Error() != "unexpected newline" {
		return fmt.Errorf("failed to read input: %w", err)
	}

	if strings.ToLower(response) != "y" {
		return nil
	}

	var (
		assigneeID  *int
		reviewerIDs []int
	)

	if suggestedAssignee != nil {
		assigneeID = &suggestedAssignee.ID
	}

	if suggestedReviewer != nil {
		reviewerIDs = []int{suggestedReviewer.ID}
	}

	err := log.WithSpinner("Applying suggestions to merge request...", func() error {
		return appInstance.UpdateMergeRequest(ctx, projectID, mrID, assigneeID, reviewerIDs)
	})
	if err != nil {
		return fmt.Errorf("failed to update merge request: %w", err)
	}

	fmt.Println("\nSuccessfully updated merge request with suggestions!")

	return nil
}

func parseMRURL(baseURL, mrURL string) (projectPath string, mrID int, err error) {
	parts := strings.Split(mrURL, "/-/merge_requests/")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid merge request URL format")
	}

	projectPath = strings.TrimPrefix(parts[0], baseURL+"/")

	mrID, err = strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid merge request ID: %w", err)
	}

	return projectPath, mrID, nil
}
