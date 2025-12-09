package cached

import (
	"context"
	"fmt"

	"github.com/denchenko/gg/internal/adapters/secondary/cache"
	"github.com/denchenko/gg/internal/adapters/secondary/repository/gitlab"
	"github.com/denchenko/gg/internal/core/app"
	"github.com/denchenko/gg/internal/core/domain"
)

// CachedRepository wraps a Repository with caching functionality.
type CachedRepository struct {
	repo  app.Repository
	cache cache.Cache
}

// NewCachedRepository creates a new cached repository instance.
func NewCachedRepository(repo app.Repository, cache cache.Cache) *CachedRepository {
	return &CachedRepository{
		repo:  repo,
		cache: cache,
	}
}

// GetProject retrieves a project by path.
func (r *CachedRepository) GetProject(ctx context.Context, path string) (*domain.Project, error) {
	project, err := r.repo.GetProject(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return project, nil
}

// ListMergeRequests lists merge requests with the given state and scope.
func (r *CachedRepository) ListMergeRequests(
	ctx context.Context,
	state string,
	scope ...string,
) ([]*domain.MergeRequest, error) {
	mrs, err := r.repo.ListMergeRequests(ctx, state, scope...)
	if err != nil {
		return nil, fmt.Errorf("failed to list merge requests: %w", err)
	}

	for _, mr := range mrs {
		if mr.Author != nil {
			r.cache.StoreUser(mr.Author)
		}
		if mr.Assignee != nil {
			r.cache.StoreUser(mr.Assignee)
		}
		for _, reviewer := range mr.Reviewers {
			if reviewer != nil {
				r.cache.StoreUser(reviewer)
			}
		}
	}

	return mrs, nil
}

// GetMergeRequestApprovals retrieves approvals for a merge request.
func (r *CachedRepository) GetMergeRequestApprovals(
	ctx context.Context,
	projectID, mrID int,
) ([]*domain.User, error) {
	users, err := r.repo.GetMergeRequestApprovals(ctx, projectID, mrID)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge request approvals: %w", err)
	}

	for _, user := range users {
		if user != nil {
			r.cache.StoreUser(user)
		}
	}

	return users, nil
}

// GetMergeRequest retrieves a merge request by project ID and MR ID.
func (r *CachedRepository) GetMergeRequest(
	ctx context.Context,
	projectID, mrID int,
) (*domain.MergeRequest, error) {
	mr, err := r.repo.GetMergeRequest(ctx, projectID, mrID)
	if err != nil {
		return nil, fmt.Errorf("failed to get merge request: %w", err)
	}

	if mr.Author != nil {
		r.cache.StoreUser(mr.Author)
	}
	if mr.Assignee != nil {
		r.cache.StoreUser(mr.Assignee)
	}
	for _, reviewer := range mr.Reviewers {
		if reviewer != nil {
			r.cache.StoreUser(reviewer)
		}
	}

	return mr, nil
}

// PreloadUsersByUsernames loads users by their usernames and caches them.
func (r *CachedRepository) PreloadUsersByUsernames(ctx context.Context, usernames []string) error {
	gitlabRepo, ok := r.repo.(*gitlab.Repository)
	if !ok {
		if err := r.repo.PreloadUsersByUsernames(ctx, usernames); err != nil {
			return fmt.Errorf("failed to preload users: %w", err)
		}

		return nil
	}

	users, err := gitlabRepo.FetchUsersByUsernames(ctx, usernames)
	if err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	for _, user := range users {
		if user != nil {
			r.cache.StoreUser(user)
		}
	}

	return nil
}

// GetAllUsers retrieves all cached users.
func (r *CachedRepository) GetAllUsers(ctx context.Context) ([]*domain.User, error) {
	cachedUsers := r.cache.GetAllUsers()
	if len(cachedUsers) > 0 {
		return cachedUsers, nil
	}

	users, err := r.repo.GetAllUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}

	return users, nil
}

// GetUserByUsername gets a user by username from cache or fetches from repository.
func (r *CachedRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	if user, ok := r.cache.GetUserByUsername(username); ok {
		return user, nil
	}

	user, err := r.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	if user != nil {
		r.cache.StoreUser(user)
	}

	return user, nil
}

// GetCurrentUser gets the current authenticated user.
func (r *CachedRepository) GetCurrentUser(ctx context.Context) (*domain.User, error) {
	user, err := r.repo.GetCurrentUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	if user != nil {
		r.cache.StoreUser(user)
	}

	return user, nil
}

// ListCommits lists commits for a project.
func (r *CachedRepository) ListCommits(ctx context.Context, projectID int) ([]*domain.Commit, error) {
	commits, err := r.repo.ListCommits(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list commits: %w", err)
	}

	return commits, nil
}

// UpdateMergeRequest updates the assignee and reviewer for a merge request.
func (r *CachedRepository) UpdateMergeRequest(
	ctx context.Context,
	projectID, mrID int,
	assigneeID *int,
	reviewerIDs []int,
) error {
	if err := r.repo.UpdateMergeRequest(ctx, projectID, mrID, assigneeID, reviewerIDs); err != nil {
		return fmt.Errorf("failed to update merge request: %w", err)
	}

	return nil
}
