package commands

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/denchenko/gg/internal/config"
	"github.com/denchenko/gg/internal/core/app"
	"github.com/denchenko/gg/internal/core/domain"
	ascii "github.com/denchenko/gg/internal/format/ascii"
	"github.com/denchenko/gg/internal/log"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
)

func MR(cfg *config.Config, appInstance *app.App, formatter *ascii.Formatter) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mr",
		Short: "Merge Requests",
	}

	cmd.AddCommand(newMRRouletteCommand(cfg, appInstance, formatter))
	cmd.AddCommand(newMRStatusCommand(cfg, appInstance, formatter))
	cmd.AddCommand(newMRBrowseCommand(appInstance))

	return cmd
}

func newMRRouletteCommand(cfg *config.Config, appInstance *app.App, formatter *ascii.Formatter) *cobra.Command {
	return &cobra.Command{
		Use:   "roulette [MR_URL]",
		Short: "Suggest assignee and reviewer for a merge request",
		Long: `Analyze team review workload and suggest appropriate assignee and reviewer for a merge request.
If MR_URL is not provided, it will try to find the merge request for the current git branch.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()
			var mrURL string

			if len(args) > 0 {
				mrURL = args[0]
			} else {
				// Infer MR from current git branch
				currentProject, currentBranch, err := appInstance.GetCurrentProjectInfo(ctx)
				if err != nil {
					return fmt.Errorf("failed to get current project info: %w", err)
				}

				mr, err := appInstance.GetMergeRequestByBranch(ctx, currentProject.ID, currentBranch)
				if err != nil {
					return fmt.Errorf("failed to find merge request for branch %s: %w", currentBranch, err)
				}

				mrURL = mr.WebURL
			}

			return suggestAssignees(cfg, appInstance, formatter, mrURL)
		},
	}
}

func newMRStatusCommand(cfg *config.Config, appInstance *app.App, formatter *ascii.Formatter) *cobra.Command {
	return &cobra.Command{
		Use:   "status [MR_URL]",
		Short: "Show status of a merge request",
		Long: `Display detailed status information for a merge request.
If MR_URL is not provided, it will try to find the merge request for the current git branch.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return showMRStatus(cfg, appInstance, formatter, args)
		},
	}
}

func showMRStatus(cfg *config.Config, appInstance *app.App, formatter *ascii.Formatter, args []string) error {
	ctx := context.Background()

	var mr *domain.MergeRequest
	var project *domain.Project
	var err error

	if len(args) > 0 {
		// Parse MR URL
		projectPath, mrID, err := parseMRURL(cfg.BaseURL, args[0])
		if err != nil {
			return fmt.Errorf("failed to parse merge request URL: %w", err)
		}

		project, err = fetchProject(ctx, appInstance, projectPath)
		if err != nil {
			return err
		}

		mr, err = fetchMergeRequest(ctx, appInstance, project.ID, mrID)
		if err != nil {
			return err
		}
	} else {
		// Infer MR from current git branch
		currentProject, currentBranch, err := appInstance.GetCurrentProjectInfo(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current project info: %w", err)
		}

		project = currentProject
		mr, err = appInstance.GetMergeRequestByBranch(ctx, project.ID, currentBranch)
		if err != nil {
			return fmt.Errorf("failed to find merge request for branch %s: %w", currentBranch, err)
		}
	}

	// Fetch approvals
	approvals, err := appInstance.GetMergeRequestApprovals(ctx, project.ID, mr.IID)
	if err != nil {
		return fmt.Errorf("failed to get merge request approvals: %w", err)
	}

	// Calculate status
	const workingDaysThreshold = 3
	now := time.Now()
	threeWorkingDaysAgo := subtractWorkingDays(now, workingDaysThreshold)
	isStalled := mr.UpdatedAt.Before(threeWorkingDaysAgo)

	mrWithStatus := &domain.MergeRequestWithStatus{
		MergeRequest:  mr,
		Approvals:     approvals,
		ApprovalCount: len(approvals),
		IsStalled:     isStalled,
	}

	// Format and display
	formatted, err := formatter.FormatMRStatus(cfg.BaseURL, mrWithStatus)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	fmt.Print(formatted)

	return nil
}

func newMRBrowseCommand(appInstance *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "browse",
		Short: "Open merge request in browser",
		Long:  `Open the merge request for the current git branch in your default browser.`,
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

			// Open URL in browser
			if err := open.Start(mr.WebURL); err != nil {
				return fmt.Errorf("failed to open browser: %w", err)
			}

			return nil
		},
	}
}

func suggestAssignees(cfg *config.Config, appInstance *app.App, formatter *ascii.Formatter, mrURL string) error {
	ctx := context.Background()

	projectPath, mrID, err := parseMRURL(cfg.BaseURL, mrURL)
	if err != nil {
		return fmt.Errorf("failed to parse merge request URL: %w", err)
	}

	project, err := fetchProject(ctx, appInstance, projectPath)
	if err != nil {
		return err
	}

	mr, err := fetchMergeRequest(ctx, appInstance, project.ID, mrID)
	if err != nil {
		return err
	}

	workloads, err := fetchWorkloads(ctx, appInstance, project.ID)
	if err != nil {
		return err
	}

	suggestedAssignee, suggestedReviewer, err := fetchSuggestions(ctx, appInstance, mr, workloads)
	if err != nil {
		return err
	}

	formatted, err := formatter.FormatMRRoulette(mr, mrURL, workloads, suggestedAssignee, suggestedReviewer)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	fmt.Print(formatted)

	if suggestedAssignee != nil || suggestedReviewer != nil {
		return applySuggestions(ctx, appInstance, project.ID, mrID, suggestedAssignee, suggestedReviewer)
	}

	return nil
}

func fetchProject(ctx context.Context, appInstance *app.App, projectPath string) (*domain.Project, error) {
	var project *domain.Project
	err := log.WithSpinner("Fetching project information...", func() error {
		var err error
		project, err = appInstance.GetProject(ctx, projectPath)
		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return project, nil
}

func fetchMergeRequest(ctx context.Context, appInstance *app.App, projectID, mrID int) (*domain.MergeRequest, error) {
	var mr *domain.MergeRequest
	err := log.WithSpinner("Fetching merge request details...", func() error {
		var err error
		mr, err = appInstance.GetMergeRequest(ctx, projectID, mrID)
		if err != nil {
			return fmt.Errorf("failed to get merge request: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get merge request: %w", err)
	}

	return mr, nil
}

func fetchWorkloads(ctx context.Context, appInstance *app.App, projectID int) ([]*domain.UserWorkload, error) {
	var workloads []*domain.UserWorkload
	err := log.WithSpinner("Analyzing team workload...", func() error {
		var err error
		workloads, err = appInstance.AnalyzeWorkload(ctx, projectID)
		if err != nil {
			return fmt.Errorf("failed to analyze workload: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to analyze workload: %w", err)
	}

	return workloads, nil
}

func fetchSuggestions(
	ctx context.Context,
	appInstance *app.App,
	mr *domain.MergeRequest,
	workloads []*domain.UserWorkload,
) (*domain.User, *domain.User, error) {
	var suggestedAssignee, suggestedReviewer *domain.User
	err := log.WithSpinner("Calculating suggestions...", func() error {
		var err error
		suggestedAssignee, suggestedReviewer, err = appInstance.SuggestAssigneeAndReviewer(ctx, mr, workloads)
		if err != nil {
			return fmt.Errorf("failed to get suggestions: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get suggestions: %w", err)
	}

	return suggestedAssignee, suggestedReviewer, nil
}

func applySuggestions(
	ctx context.Context,
	appInstance *app.App,
	projectID, mrID int,
	suggestedAssignee, suggestedReviewer *domain.User,
) error {
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

const urlPartsCount = 2

func parseMRURL(baseURL, mrURL string) (projectPath string, mrID int, err error) {
	parts := strings.Split(mrURL, "/-/merge_requests/")
	if len(parts) != urlPartsCount {
		return "", 0, errors.New("invalid merge request URL format")
	}

	projectPath = strings.TrimPrefix(parts[0], baseURL+"/")

	mrID, err = strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid merge request ID: %w", err)
	}

	return projectPath, mrID, nil
}

func subtractWorkingDays(date time.Time, days int) time.Time {
	result := date
	subtractedDays := 0

	for subtractedDays < days {
		result = result.AddDate(0, 0, -1)
		if result.Weekday() != time.Saturday && result.Weekday() != time.Sunday {
			subtractedDays++
		}
	}

	return result
}
