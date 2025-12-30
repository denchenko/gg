package app

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/denchenko/gg/internal/config"
	"github.com/denchenko/gg/internal/core/domain"
	"golang.org/x/sync/errgroup"
)

const workingDaysThreshold = 3

// Repository defines the interface for data persistence operations (port).
type Repository interface {
	GetProject(ctx context.Context, path string) (*domain.Project, error)
	ListMergeRequests(ctx context.Context, state string, scope ...string) ([]*domain.MergeRequest, error)
	GetMergeRequestApprovals(ctx context.Context, projectID, mrID int) ([]*domain.User, error)
	GetMergeRequest(ctx context.Context, projectID, mrID int) (*domain.MergeRequest, error)
	PreloadUsersByUsernames(ctx context.Context, usernames []string) error
	GetAllUsers(ctx context.Context) ([]*domain.User, error)
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)
	GetCurrentUser(ctx context.Context) (*domain.User, error)
	ListCommits(ctx context.Context, projectID int) ([]*domain.Commit, error)
	UpdateMergeRequest(ctx context.Context, projectID, mrID int, assigneeID *int, reviewerIDs []int) error
	GetUserEvents(ctx context.Context, userID int, since time.Time, till *time.Time) ([]*domain.Event, error)
}

// App represents the core application with all business logic.
type App struct {
	repo      Repository
	teamUsers []string
}

// NewApp creates a new application instance.
func NewApp(cfg *config.Config, repo Repository) (*App, error) {
	ctx := context.Background()

	if err := repo.PreloadUsersByUsernames(ctx, cfg.TeamUsers); err != nil {
		fmt.Printf("Warning: failed to preload users: %v\n", err)
	}

	return &App{
		repo:      repo,
		teamUsers: cfg.TeamUsers,
	}, nil
}

// AnalyzeWorkload analyzes the workload for team members.
func (a *App) AnalyzeWorkload(ctx context.Context, projectID int) ([]*domain.UserWorkload, error) {
	emailToUserID, err := a.buildEmailToUserIDMap(ctx)
	if err != nil {
		return nil, err
	}

	userCommits, err := a.countUserCommits(ctx, projectID, emailToUserID)
	if err != nil {
		return nil, err
	}

	mrs, err := a.repo.ListMergeRequests(ctx, "opened", "all")
	if err != nil {
		return nil, fmt.Errorf("failed to get merge requests: %w", err)
	}

	workloads := make([]*domain.UserWorkload, 0, len(a.teamUsers))
	for _, username := range a.teamUsers {
		user, err := a.repo.GetUserByUsername(ctx, username)
		if err != nil {
			continue
		}

		activeMRCount := 0
		for _, mr := range mrs {
			if !isUserInvolvedInMR(mr, user.ID) {
				continue
			}

			approvals, err := a.repo.GetMergeRequestApprovals(ctx, mr.ProjectID, mr.IID)
			if err != nil {
				continue
			}

			if !hasUserApprovedMR(approvals, user.ID) {
				activeMRCount++
			}
		}

		workloads = append(workloads, &domain.UserWorkload{
			User:    user,
			MRCount: activeMRCount,
			Commits: userCommits[user.ID],
		})
	}

	return workloads, nil
}

// AnalyzeActiveMRs analyzes active merge requests for team members.
func (a *App) AnalyzeActiveMRs(ctx context.Context) ([]*domain.UserWorkload, error) {
	mrs, err := a.repo.ListMergeRequests(ctx, "opened", "all")
	if err != nil {
		return nil, fmt.Errorf("failed to get merge requests: %w", err)
	}

	workloads := make([]*domain.UserWorkload, 0, len(a.teamUsers))
	for _, username := range a.teamUsers {
		user, err := a.repo.GetUserByUsername(ctx, username)
		if err != nil {
			continue
		}

		var relevantMRs []*domain.MergeRequest
		for _, mr := range mrs {
			if isUserInvolvedInMR(mr, user.ID) {
				relevantMRs = append(relevantMRs, mr)
			}
		}

		approvalsMap, err := a.fetchMRApprovals(ctx, relevantMRs)
		if err != nil {
			continue
		}

		activeMRCount := 0
		var activeMRs []*domain.MergeRequest
		for _, mr := range relevantMRs {
			if !hasUserApprovedMR(approvalsMap[mr.IID], user.ID) {
				activeMRCount++
				activeMRs = append(activeMRs, mr)
			}
		}

		workloads = append(workloads, &domain.UserWorkload{
			User:      user,
			MRCount:   activeMRCount,
			ActiveMRs: activeMRs,
		})
	}

	return workloads, nil
}

// AnalyzeMyReviewWorkload analyzes the current user's review workload.
func (a *App) AnalyzeMyReviewWorkload(ctx context.Context) (*domain.UserWorkload, error) {
	currentUser, err := a.getCurrentUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	// Get all opened merge requests
	mrs, err := a.repo.ListMergeRequests(ctx, "opened", "all")
	if err != nil {
		return nil, fmt.Errorf("failed to get merge requests: %w", err)
	}

	var relevantMRs []*domain.MergeRequest
	for _, mr := range mrs {
		if isUserInvolvedInMR(mr, currentUser.ID) {
			relevantMRs = append(relevantMRs, mr)
		}
	}

	// Get approvals for relevant MRs
	approvalsMap, err := a.fetchMRApprovals(ctx, relevantMRs)
	if err != nil {
		return nil, fmt.Errorf("failed to get MR approvals: %w", err)
	}

	activeMRCount := 0
	var activeMRs []*domain.MergeRequest
	for _, mr := range relevantMRs {
		if !hasUserApprovedMR(approvalsMap[mr.IID], currentUser.ID) {
			activeMRCount++
			activeMRs = append(activeMRs, mr)
		}
	}

	return &domain.UserWorkload{
		User:      currentUser,
		MRCount:   activeMRCount,
		ActiveMRs: activeMRs,
	}, nil
}

func (a *App) getCurrentUser(ctx context.Context) (*domain.User, error) {
	user, err := a.repo.GetCurrentUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	return user, nil
}

// SuggestAssigneeAndReviewer suggests an assignee and reviewer for a merge request.
func (a *App) SuggestAssigneeAndReviewer(
	_ context.Context,
	mr *domain.MergeRequest,
	workloads []*domain.UserWorkload,
) (*domain.User, *domain.User, error) {
	if len(workloads) == 0 {
		return nil, nil, errors.New("no team members available")
	}

	availableWorkloads := make([]*domain.UserWorkload, 0, len(workloads))
	for _, workload := range workloads {
		if isUserAvailable(workload.User) {
			availableWorkloads = append(availableWorkloads, workload)
		}
	}

	if len(availableWorkloads) == 0 {
		return nil, nil, errors.New("no available team members")
	}

	sort.Slice(availableWorkloads, func(i, j int) bool {
		scoreI := calculateAssigneeScore(availableWorkloads[i].Commits, availableWorkloads[i].MRCount)
		scoreJ := calculateAssigneeScore(availableWorkloads[j].Commits, availableWorkloads[j].MRCount)

		return scoreI > scoreJ
	})

	var suggestedAssignee *domain.User
	for _, workload := range availableWorkloads {
		if mr.Author == nil || workload.User.ID != mr.Author.ID {
			suggestedAssignee = workload.User

			break
		}
	}

	sort.Slice(availableWorkloads, func(i, j int) bool {
		return availableWorkloads[i].MRCount < availableWorkloads[j].MRCount
	})

	var suggestedReviewer *domain.User
	for _, workload := range availableWorkloads {
		if (mr.Author == nil || workload.User.ID != mr.Author.ID) &&
			(suggestedAssignee == nil || workload.User.ID != suggestedAssignee.ID) {
			suggestedReviewer = workload.User

			break
		}
	}

	return suggestedAssignee, suggestedReviewer, nil
}

// GetProject retrieves a project by path.
func (a *App) GetProject(ctx context.Context, projectPath string) (*domain.Project, error) {
	project, err := a.repo.GetProject(ctx, projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return project, nil
}

// GetMergeRequest retrieves a merge request by project ID and MR ID.
func (a *App) GetMergeRequest(ctx context.Context, projectID, mrID int) (*domain.MergeRequest, error) {
	mr, err := a.repo.GetMergeRequest(ctx, projectID, mrID)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge request: %w", err)
	}

	return mr, nil
}

// GetMergeRequestByBranch retrieves a merge request by project ID and source branch.
func (a *App) GetMergeRequestByBranch(ctx context.Context, projectID int, branch string) (*domain.MergeRequest, error) {
	mrs, err := a.repo.ListMergeRequests(ctx, "opened", "all")
	if err != nil {
		return nil, fmt.Errorf("failed to list merge requests: %w", err)
	}

	var matchingMRs []*domain.MergeRequest
	for _, mr := range mrs {
		if mr.ProjectID == projectID && mr.SourceBranch == branch {
			matchingMRs = append(matchingMRs, mr)
		}
	}

	if len(matchingMRs) == 0 {
		return nil, fmt.Errorf("no merge request found for branch %s", branch)
	}

	if len(matchingMRs) > 1 {
		// If multiple MRs match, return the most recently updated one
		sort.Slice(matchingMRs, func(i, j int) bool {
			return matchingMRs[i].UpdatedAt.After(matchingMRs[j].UpdatedAt)
		})
	}

	return matchingMRs[0], nil
}

// ListMergeRequests lists merge requests with the given state and scope.
func (a *App) ListMergeRequests(ctx context.Context, state string, scope ...string) ([]*domain.MergeRequest, error) {
	mrs, err := a.repo.ListMergeRequests(ctx, state, scope...)
	if err != nil {
		return nil, fmt.Errorf("failed to list merge requests: %w", err)
	}

	return mrs, nil
}

// GetMergeRequestApprovals retrieves approvals for a merge request.
func (a *App) GetMergeRequestApprovals(ctx context.Context, projectID, mrID int) ([]*domain.User, error) {
	approvals, err := a.repo.GetMergeRequestApprovals(ctx, projectID, mrID)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge request approvals: %w", err)
	}

	return approvals, nil
}

// GetCurrentProjectInfo retrieves information about the current project from git.
func (a *App) GetCurrentProjectInfo(ctx context.Context) (*domain.Project, string, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get remote URL: %w", err)
	}

	remoteURL := strings.TrimSpace(string(output))

	// Get current branch
	cmd = exec.CommandContext(ctx, "git", "branch", "--show-current")
	output, err = cmd.Output()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return nil, "", errors.New("failed to get current branch")
	}

	const urlPartsCount = 2

	var projectPath string
	if strings.HasPrefix(remoteURL, "git@") {
		parts := strings.Split(strings.TrimSuffix(remoteURL, ".git"), ":")
		if len(parts) != urlPartsCount {
			return nil, "", errors.New("invalid SSH remote URL format")
		}
		projectPath = parts[1]
	} else {
		parts := strings.Split(strings.TrimSuffix(remoteURL, ".git"), "/")
		if len(parts) < urlPartsCount {
			return nil, "", errors.New("invalid HTTPS remote URL format")
		}
		projectPath = strings.Join(parts[len(parts)-2:], "/")
	}

	project, err := a.repo.GetProject(ctx, projectPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get project: %w", err)
	}

	return project, branch, nil
}

// SortMergeRequestsByPriority sorts merge requests by priority.
func (a *App) SortMergeRequestsByPriority(
	mrs []*domain.MergeRequestWithStatus,
	_ int,
	_ string,
) []*domain.MergeRequestWithStatus {
	sorted := make([]*domain.MergeRequestWithStatus, len(mrs))
	copy(sorted, mrs)

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].IsCurrentBranch && !sorted[j].IsCurrentBranch {
			return true
		}
		if !sorted[i].IsCurrentBranch && sorted[j].IsCurrentBranch {
			return false
		}

		if sorted[i].IsCurrentProject && !sorted[j].IsCurrentProject {
			return true
		}
		if !sorted[i].IsCurrentProject && sorted[j].IsCurrentProject {
			return false
		}

		return sorted[i].UpdatedAt.After(sorted[j].UpdatedAt)
	})

	return sorted
}

// GetMergeRequestsWithStatus retrieves merge requests with enhanced status information.
func (a *App) GetMergeRequestsWithStatus(ctx context.Context) ([]*domain.MergeRequestWithStatus, error) {
	mrs, err := a.repo.ListMergeRequests(ctx, "opened")
	if err != nil {
		return nil, fmt.Errorf("failed to get merge requests: %w", err)
	}

	// Try to get current project info (may fail if not in git repo)
	var currentProjectID int
	var currentBranch string
	currentProject, currentBranch, err := a.GetCurrentProjectInfo(ctx)
	if err == nil {
		currentProjectID = currentProject.ID
	}

	mrsWithStatus := make([]*domain.MergeRequestWithStatus, 0, len(mrs))
	now := time.Now()
	threeWorkingDaysAgo := subtractWorkingDays(now, workingDaysThreshold)

	for _, mr := range mrs {
		approvals, err := a.repo.GetMergeRequestApprovals(ctx, mr.ProjectID, mr.IID)
		if err != nil {
			approvals = []*domain.User{}
		}

		isStalled := mr.UpdatedAt.Before(threeWorkingDaysAgo)
		isCurrentBranch := currentBranch != "" && mr.SourceBranch == currentBranch
		isCurrentProject := currentProjectID != 0 && mr.ProjectID == currentProjectID

		mrWithStatus := &domain.MergeRequestWithStatus{
			MergeRequest:     mr,
			Approvals:        approvals,
			ApprovalCount:    len(approvals),
			IsStalled:        isStalled,
			IsCurrentBranch:  isCurrentBranch,
			IsCurrentProject: isCurrentProject,
		}

		mrsWithStatus = append(mrsWithStatus, mrWithStatus)
	}

	return a.SortMergeRequestsByPriority(mrsWithStatus, currentProjectID, currentBranch), nil
}

// GetCurrentMRURL retrieves the current merge request URL from git.
func (a *App) GetCurrentMRURL(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}

	remoteURL := strings.TrimSpace(string(output))

	cmd = exec.CommandContext(ctx, "git", "branch", "--show-current")
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "", errors.New("failed to get current branch")
	}

	if strings.HasPrefix(remoteURL, "git@") {
		remoteURL = strings.Replace(remoteURL, "git@gitlab.com:", "https://gitlab.com/", 1)
		remoteURL = strings.Replace(remoteURL, ".git", "", 1)
	}

	return fmt.Sprintf("%s/-/merge_requests/new?merge_request[source_branch]=%s", remoteURL, branch), nil
}

// UpdateMergeRequest updates a merge request with optional assignee and reviewers.
func (a *App) UpdateMergeRequest(ctx context.Context, projectID, mrID int, assigneeID *int, reviewerIDs []int) error {
	if err := a.repo.UpdateMergeRequest(ctx, projectID, mrID, assigneeID, reviewerIDs); err != nil {
		return fmt.Errorf("failed to update merge request: %w", err)
	}

	return nil
}

// GetMyReviewWorkloadWithStatus retrieves merge requests with enhanced status information
// for current user's review workload.
func (a *App) GetMyReviewWorkloadWithStatus(ctx context.Context) ([]*domain.MergeRequestWithStatus, error) {
	currentUser, err := a.getCurrentUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	mrs, err := a.repo.ListMergeRequests(ctx, "opened", "all")
	if err != nil {
		return nil, fmt.Errorf("failed to get merge requests: %w", err)
	}

	currentProjectID, currentBranch := a.getCurrentProjectInfoSafe(ctx)
	threeWorkingDaysAgo := subtractWorkingDays(time.Now(), workingDaysThreshold)

	mrsWithStatus := a.filterAndEnrichMRsForReview(
		ctx, mrs, currentUser, currentProjectID, currentBranch, threeWorkingDaysAgo,
	)

	return a.SortMergeRequestsByPriority(mrsWithStatus, currentProjectID, currentBranch), nil
}

// GetMyActivity retrieves the current user's activity events within the specified time range.
func (a *App) GetMyActivity(ctx context.Context, since time.Time, till *time.Time) ([]*domain.Event, error) {
	currentUser, err := a.getCurrentUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	events, err := a.repo.GetUserEvents(ctx, currentUser.ID, since, till)
	if err != nil {
		return nil, fmt.Errorf("failed to get user events: %w", err)
	}

	return events, nil
}

func (a *App) getCurrentProjectInfoSafe(ctx context.Context) (int, string) {
	currentProject, currentBranch, err := a.GetCurrentProjectInfo(ctx)
	if err != nil {
		return 0, ""
	}

	return currentProject.ID, currentBranch
}

func (a *App) filterAndEnrichMRsForReview(
	ctx context.Context,
	mrs []*domain.MergeRequest,
	currentUser *domain.User,
	currentProjectID int,
	currentBranch string,
	threeWorkingDaysAgo time.Time,
) []*domain.MergeRequestWithStatus {
	mrsWithStatus := make([]*domain.MergeRequestWithStatus, 0)

	for _, mr := range mrs {
		if !a.isMRRelevantForReview(mr, currentUser) {
			continue
		}

		approvals, _ := a.repo.GetMergeRequestApprovals(ctx, mr.ProjectID, mr.IID)
		if hasUserApprovedMR(approvals, currentUser.ID) {
			continue
		}

		mrWithStatus := a.createMRWithStatus(mr, approvals, currentProjectID, currentBranch, threeWorkingDaysAgo)
		mrsWithStatus = append(mrsWithStatus, mrWithStatus)
	}

	return mrsWithStatus
}

func (a *App) isMRRelevantForReview(mr *domain.MergeRequest, currentUser *domain.User) bool {
	if mr.Draft {
		return false
	}

	if mr.Author != nil && mr.Author.ID == currentUser.ID {
		return false
	}

	isAssignee := mr.Assignee != nil && mr.Assignee.ID == currentUser.ID
	isReviewer := false
	for _, reviewer := range mr.Reviewers {
		if reviewer.ID == currentUser.ID {
			isReviewer = true

			break
		}
	}

	return isAssignee || isReviewer
}

func (a *App) createMRWithStatus(
	mr *domain.MergeRequest,
	approvals []*domain.User,
	currentProjectID int,
	currentBranch string,
	threeWorkingDaysAgo time.Time,
) *domain.MergeRequestWithStatus {
	isStalled := mr.UpdatedAt.Before(threeWorkingDaysAgo)
	isCurrentBranch := currentBranch != "" && mr.SourceBranch == currentBranch
	isCurrentProject := currentProjectID != 0 && mr.ProjectID == currentProjectID

	return &domain.MergeRequestWithStatus{
		MergeRequest:     mr,
		Approvals:        approvals,
		ApprovalCount:    len(approvals),
		IsStalled:        isStalled,
		IsCurrentBranch:  isCurrentBranch,
		IsCurrentProject: isCurrentProject,
	}
}

func calculateAssigneeScore(commits, mrCount int) float64 {
	return float64(commits) / (1 + float64(mrCount))
}

func (a *App) buildEmailToUserIDMap(ctx context.Context) (map[string]int, error) {
	allUsers, err := a.repo.GetAllUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}

	emailToUserID := make(map[string]int)
	for _, user := range allUsers {
		if user.Email != "" {
			emailToUserID[user.Email] = user.ID
		}
	}

	return emailToUserID, nil
}

func (a *App) countUserCommits(ctx context.Context, projectID int, emailToUserID map[string]int) (map[int]int, error) {
	commits, err := a.repo.ListCommits(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}

	userCommits := make(map[int]int)
	for _, commit := range commits {
		if commit.AuthorEmail == "" {
			continue
		}

		userID, exists := emailToUserID[commit.AuthorEmail]
		if !exists {
			username := strings.Split(commit.AuthorEmail, "@")[0]
			user, err := a.repo.GetUserByUsername(ctx, username)
			if err != nil {
				continue
			}
			userID = user.ID
			emailToUserID[commit.AuthorEmail] = userID
		}

		userCommits[userID]++
	}

	return userCommits, nil
}

func isUserInvolvedInMR(mr *domain.MergeRequest, userID int) bool {
	if mr == nil {
		return false
	}
	if mr.Draft {
		return false
	}
	if mr.Author != nil && mr.Author.ID == userID {
		return false
	}
	isAssignee := mr.Assignee != nil && mr.Assignee.ID == userID
	isReviewer := false
	for _, reviewer := range mr.Reviewers {
		if reviewer.ID == userID {
			isReviewer = true

			break
		}
	}

	return isAssignee || isReviewer
}

func (a *App) fetchMRApprovals(ctx context.Context, mrs []*domain.MergeRequest) (map[int][]*domain.User, error) {
	g, ctx := errgroup.WithContext(ctx)
	approvalsMap := make(map[int][]*domain.User, len(mrs))
	var mu sync.Mutex

	for _, mr := range mrs {
		g.Go(func() error {
			approvals, err := a.repo.GetMergeRequestApprovals(ctx, mr.ProjectID, mr.IID)
			if err != nil {
				return fmt.Errorf("failed to get approvals for MR %d: %w", mr.IID, err)
			}
			mu.Lock()
			approvalsMap[mr.IID] = approvals
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("failed to fetch MR approvals: %w", err)
	}

	return approvalsMap, nil
}

func hasUserApprovedMR(approvals []*domain.User, userID int) bool {
	for _, approval := range approvals {
		if approval.ID == userID {
			return true
		}
	}

	return false
}

func isUserAvailable(user *domain.User) bool {
	status := strings.ToLower(user.Status.Message)
	availability := strings.ToLower(user.Status.Availability)

	return !strings.Contains(status, "ooo") &&
		!strings.Contains(status, "vacation") &&
		availability != "busy"
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
