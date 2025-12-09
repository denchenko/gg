package gitlab

import (
	"context"
	"fmt"
	"sync"

	"github.com/denchenko/gg/internal/core/domain"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"golang.org/x/sync/errgroup"
)

const perPageLimit = 100

// Repository implements the app.Repository interface for GitLab.
type Repository struct {
	client *gitlab.Client
}

// NewRepository creates a new GitLab repository instance.
func NewRepository(client *gitlab.Client) *Repository {
	return &Repository{
		client: client,
	}
}

// PreloadUsersByUsernames loads users by their usernames.
func (r *Repository) PreloadUsersByUsernames(_ context.Context, _ []string) error {
	return nil
}

// FetchUsersByUsernames fetches users by their usernames from the GitLab API.
func (r *Repository) FetchUsersByUsernames(ctx context.Context, usernames []string) ([]*domain.User, error) {
	usernameMap := make(map[string]struct{}, len(usernames))
	for _, username := range usernames {
		usernameMap[username] = struct{}{}
	}

	users, _, err := r.client.Users.ListUsers(&gitlab.ListUsersOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: perPageLimit,
		},
		Active: pointerOf(true),
		Humans: pointerOf(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	var errg errgroup.Group
	var mu sync.Mutex
	domainUsers := make([]*domain.User, 0, len(usernames))

	for _, user := range users {
		if _, ok := usernameMap[user.Username]; ok {
			user := user // capture loop variable
			errg.Go(func() error {
				domainUser := &domain.User{
					ID:       user.ID,
					Username: user.Username,
					Email:    user.Email,
				}
				domainUser.Status, err = r.getUserStatus(ctx, user.ID)
				if err != nil {
					return fmt.Errorf("failed to get user status: %w", err)
				}
				mu.Lock()
				domainUsers = append(domainUsers, domainUser)
				mu.Unlock()

				return nil
			})
		}
	}

	if err := errg.Wait(); err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	return domainUsers, nil
}

// getUserStatus gets a user's status from GitLab.
func (r *Repository) getUserStatus(_ context.Context, userID int) (domain.UserStatus, error) {
	status, _, err := r.client.Users.GetUserStatus(userID)
	if err != nil {
		return domain.UserStatus{}, fmt.Errorf("failed to get user status: %w", err)
	}

	return domain.UserStatus{
		Message:      status.Message,
		Availability: string(status.Availability),
	}, nil
}

// getUser fetches a user by ID from the GitLab API.
func (r *Repository) getUser(ctx context.Context, userID int) (*domain.User, error) {
	user, _, err := r.client.Users.GetUser(userID, gitlab.GetUsersOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	domainUser := &domain.User{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
	}

	domainUser.Status, err = r.getUserStatus(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user status: %w", err)
	}

	return domainUser, nil
}

// GetUserByUsername gets a user by username.
func (r *Repository) GetUserByUsername(_ context.Context, username string) (*domain.User, error) {
	return nil, fmt.Errorf("user not found: %s (GitLab API doesn't support fetching by username)", username)
}

// GetCurrentUser gets the current authenticated user.
func (r *Repository) GetCurrentUser(ctx context.Context) (*domain.User, error) {
	user, _, err := r.client.Users.CurrentUser()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	domainUser := &domain.User{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
	}

	domainUser.Status, err = r.getUserStatus(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user status: %w", err)
	}

	return domainUser, nil
}

func (r *Repository) batchGetUsers(ctx context.Context, userIDs []int) (map[int]*domain.User, error) {
	var wg sync.WaitGroup
	users := make(map[int]*domain.User)
	var mu sync.Mutex
	errChan := make(chan error, len(userIDs))

	for _, userID := range userIDs {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			user, err := r.getUser(ctx, id)
			if err != nil {
				errChan <- err

				return
			}
			mu.Lock()
			users[id] = user
			mu.Unlock()
		}(userID)
	}

	wg.Wait()
	close(errChan)

	if err := <-errChan; err != nil {
		return nil, err
	}

	return users, nil
}

// GetProject retrieves a project by path.
func (r *Repository) GetProject(_ context.Context, path string) (*domain.Project, error) {
	project, _, err := r.client.Projects.GetProject(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &domain.Project{
		ID:   project.ID,
		Path: project.Path,
	}, nil
}

// GetMergeRequest retrieves a merge request by project ID and MR ID.
func (r *Repository) GetMergeRequest(ctx context.Context, projectID, mrID int) (*domain.MergeRequest, error) {
	mr, _, err := r.client.MergeRequests.GetMergeRequest(projectID, mrID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge request: %w", err)
	}

	userIDs := []int{mr.Author.ID}
	if mr.Assignee != nil {
		userIDs = append(userIDs, mr.Assignee.ID)
	}
	for _, reviewer := range mr.Reviewers {
		userIDs = append(userIDs, reviewer.ID)
	}

	users, err := r.batchGetUsers(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	reviewers := make([]*domain.User, 0, len(mr.Reviewers))
	for _, reviewer := range mr.Reviewers {
		if user, ok := users[reviewer.ID]; ok {
			reviewers = append(reviewers, user)
		}
	}

	domainMR := &domain.MergeRequest{
		ID:           mr.ID,
		IID:          mr.IID,
		Title:        mr.Title,
		Description:  mr.Description,
		WebURL:       mr.WebURL,
		Author:       users[mr.Author.ID],
		Assignee:     nil,
		Reviewers:    reviewers,
		CreatedAt:    *mr.CreatedAt,
		UpdatedAt:    *mr.UpdatedAt,
		ProjectID:    mr.ProjectID,
		Draft:        mr.Draft,
		SourceBranch: mr.SourceBranch,
	}

	if mr.Assignee != nil {
		if assignee, ok := users[mr.Assignee.ID]; ok {
			domainMR.Assignee = assignee
		}
	}

	return domainMR, nil
}

// ListMergeRequests lists merge requests with the given state and scope.
func (r *Repository) ListMergeRequests(
	ctx context.Context,
	state string,
	scope ...string,
) ([]*domain.MergeRequest, error) {
	opts := &gitlab.ListMergeRequestsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: perPageLimit,
		},
		State: &state,
	}
	if len(scope) > 0 {
		opts.Scope = &scope[0]
	}
	mrs, _, err := r.client.MergeRequests.ListMergeRequests(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list merge requests: %w", err)
	}

	userIDs := make(map[int]struct{})
	for _, mr := range mrs {
		userIDs[mr.Author.ID] = struct{}{}
		if mr.Assignee != nil {
			userIDs[mr.Assignee.ID] = struct{}{}
		}
		for _, reviewer := range mr.Reviewers {
			userIDs[reviewer.ID] = struct{}{}
		}
	}

	userIDSlice := make([]int, 0, len(userIDs))
	for id := range userIDs {
		userIDSlice = append(userIDSlice, id)
	}

	users, err := r.batchGetUsers(ctx, userIDSlice)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	return r.convertToDomainMRs(mrs, users), nil
}

// GetMergeRequestApprovals retrieves approvals for a merge request.
func (r *Repository) GetMergeRequestApprovals(ctx context.Context, projectID, mrID int) ([]*domain.User, error) {
	approvals, _, err := r.client.MergeRequests.GetMergeRequestApprovals(projectID, mrID)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge request approvals: %w", err)
	}

	users := make([]*domain.User, 0, len(approvals.ApprovedBy))
	for _, approval := range approvals.ApprovedBy {
		user, err := r.getUser(ctx, approval.User.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get approval user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// GetUser retrieves a user by ID (not part of Repository interface but used internally).
func (r *Repository) GetUser(ctx context.Context, userID int) (*domain.User, error) {
	user, _, err := r.client.Users.GetUser(userID, gitlab.GetUsersOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	domainUser := &domain.User{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
	}

	domainUser.Status, err = r.getUserStatus(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user status: %w", err)
	}

	return domainUser, nil
}

// GetAllUsers retrieves all users.
func (r *Repository) GetAllUsers(_ context.Context) ([]*domain.User, error) {
	return []*domain.User{}, nil
}

// ListCommits lists commits for a project.
func (r *Repository) ListCommits(_ context.Context, projectID int) ([]*domain.Commit, error) {
	commits, _, err := r.client.Commits.ListCommits(projectID, &gitlab.ListCommitsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: perPageLimit,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list commits: %w", err)
	}

	result := make([]*domain.Commit, 0, len(commits))
	for _, commit := range commits {
		result = append(result, &domain.Commit{
			ID:          commit.ID,
			AuthorName:  commit.AuthorName,
			AuthorEmail: commit.AuthorEmail,
			CreatedAt:   *commit.CreatedAt,
			Message:     commit.Message,
			WebURL:      commit.WebURL,
		})
	}

	return result, nil
}

// UpdateMergeRequest updates the assignee and reviewer for a merge request.
func (r *Repository) UpdateMergeRequest(
	_ context.Context,
	projectID, mrID int,
	assigneeID *int,
	reviewerIDs []int,
) error {
	opts := &gitlab.UpdateMergeRequestOptions{}

	if assigneeID != nil {
		opts.AssigneeID = assigneeID
	}

	if len(reviewerIDs) > 0 {
		opts.ReviewerIDs = &reviewerIDs
	}

	_, _, err := r.client.MergeRequests.UpdateMergeRequest(projectID, mrID, opts)
	if err != nil {
		return fmt.Errorf("failed to update merge request: %w", err)
	}

	return nil
}

func (r *Repository) convertToDomainMRs(
	mrs []*gitlab.BasicMergeRequest,
	users map[int]*domain.User,
) []*domain.MergeRequest {
	domainMRs := make([]*domain.MergeRequest, 0, len(mrs))
	for _, mr := range mrs {
		reviewers := make([]*domain.User, 0, len(mr.Reviewers))
		for _, reviewer := range mr.Reviewers {
			if user, ok := users[reviewer.ID]; ok {
				reviewers = append(reviewers, user)
			}
		}

		domainMR := &domain.MergeRequest{
			ID:           mr.ID,
			IID:          mr.IID,
			Title:        mr.Title,
			Description:  mr.Description,
			WebURL:       mr.WebURL,
			Author:       users[mr.Author.ID],
			Assignee:     nil,
			Reviewers:    reviewers,
			CreatedAt:    *mr.CreatedAt,
			UpdatedAt:    *mr.UpdatedAt,
			ProjectID:    mr.ProjectID,
			Draft:        mr.Draft,
			SourceBranch: mr.SourceBranch,
		}

		if mr.Assignee != nil {
			if assignee, ok := users[mr.Assignee.ID]; ok {
				domainMR.Assignee = assignee
			}
		}

		domainMRs = append(domainMRs, domainMR)
	}

	return domainMRs
}

func pointerOf(v bool) *bool {
	return &v
}
