package mocks

import (
	"context"
	"time"

	"github.com/denchenko/gg/internal/core/domain"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of app.Repository.
type MockRepository struct {
	mock.Mock
}

// GetProject mocks the GetProject method.
func (m *MockRepository) GetProject(ctx context.Context, path string) (*domain.Project, error) {
	args := m.Called(ctx, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*domain.Project), args.Error(1)
}

// ListMergeRequests mocks the ListMergeRequests method.
func (m *MockRepository) ListMergeRequests(
	ctx context.Context,
	state string,
	scope ...string,
) ([]*domain.MergeRequest, error) {
	// Convert variadic to slice for mock matching
	scopeSlice := []string{}
	if len(scope) > 0 {
		scopeSlice = scope
	}
	args := m.Called(ctx, state, scopeSlice)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]*domain.MergeRequest), args.Error(1)
}

// GetMergeRequestApprovals mocks the GetMergeRequestApprovals method.
func (m *MockRepository) GetMergeRequestApprovals(ctx context.Context, projectID, mrID int) ([]*domain.User, error) {
	args := m.Called(ctx, projectID, mrID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]*domain.User), args.Error(1)
}

// GetMergeRequest mocks the GetMergeRequest method.
func (m *MockRepository) GetMergeRequest(ctx context.Context, projectID, mrID int) (*domain.MergeRequest, error) {
	args := m.Called(ctx, projectID, mrID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*domain.MergeRequest), args.Error(1)
}

// PreloadUsersByUsernames mocks the PreloadUsersByUsernames method.
func (m *MockRepository) PreloadUsersByUsernames(ctx context.Context, usernames []string) error {
	args := m.Called(ctx, usernames)

	return args.Error(0)
}

// GetAllUsers mocks the GetAllUsers method.
func (m *MockRepository) GetAllUsers(ctx context.Context) ([]*domain.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]*domain.User), args.Error(1)
}

// GetUserByUsername mocks the GetUserByUsername method.
func (m *MockRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*domain.User), args.Error(1)
}

// GetCurrentUser mocks the GetCurrentUser method.
func (m *MockRepository) GetCurrentUser(ctx context.Context) (*domain.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*domain.User), args.Error(1)
}

// ListCommits mocks the ListCommits method.
func (m *MockRepository) ListCommits(ctx context.Context, projectID int) ([]*domain.Commit, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]*domain.Commit), args.Error(1)
}

// UpdateMergeRequest mocks the UpdateMergeRequest method.
func (m *MockRepository) UpdateMergeRequest(
	ctx context.Context,
	projectID, mrID int,
	assigneeID *int,
	reviewerIDs []int,
) error {
	args := m.Called(ctx, projectID, mrID, assigneeID, reviewerIDs)

	return args.Error(0)
}

// GetUserEvents mocks the GetUserEvents method.
func (m *MockRepository) GetUserEvents(
	ctx context.Context,
	userID int,
	since time.Time,
	till *time.Time,
) ([]*domain.Event, error) {
	args := m.Called(ctx, userID, since, till)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]*domain.Event), args.Error(1)
}
