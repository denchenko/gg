package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/denchenko/gg/internal/adapters/secondary/repository/mocks"
	"github.com/denchenko/gg/internal/config"
	"github.com/denchenko/gg/internal/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewApp(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		repo        *mocks.MockRepository
		expectError bool
		setupMock   func(*mocks.MockRepository)
	}{
		{
			name: "successful creation",
			cfg: &config.Config{
				TeamUsers: []string{"user1", "user2"},
			},
			repo:        &mocks.MockRepository{},
			expectError: false,
			setupMock: func(m *mocks.MockRepository) {
				m.On("PreloadUsersByUsernames", mock.Anything, []string{"user1", "user2"}).Return(nil)
			},
		},
		{
			name: "preload error is ignored",
			cfg: &config.Config{
				TeamUsers: []string{"user1"},
			},
			repo:        &mocks.MockRepository{},
			expectError: false,
			setupMock: func(m *mocks.MockRepository) {
				m.On("PreloadUsersByUsernames", mock.Anything, []string{"user1"}).Return(errors.New("preload failed"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(tt.repo)

			app, err := NewApp(tt.cfg, tt.repo)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, app)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, app)
				assert.Equal(t, tt.cfg.TeamUsers, app.teamUsers)
			}

			tt.repo.AssertExpectations(t)
		})
	}
}

func TestApp_GetProject(t *testing.T) {
	ctx := context.Background()
	repo := &mocks.MockRepository{}
	app := &App{repo: repo, teamUsers: []string{}}

	expectedProject := &domain.Project{
		ID:   1,
		Path: "group/project",
	}

	repo.On("GetProject", ctx, "group/project").Return(expectedProject, nil)

	project, err := app.GetProject(ctx, "group/project")

	require.NoError(t, err)
	assert.Equal(t, expectedProject, project)
	repo.AssertExpectations(t)
}

func TestApp_GetMergeRequest(t *testing.T) {
	ctx := context.Background()
	repo := &mocks.MockRepository{}
	app := &App{repo: repo, teamUsers: []string{}}

	expectedMR := &domain.MergeRequest{
		ID:    1,
		IID:   2,
		Title: "Test MR",
	}

	repo.On("GetMergeRequest", ctx, 1, 2).Return(expectedMR, nil)

	mr, err := app.GetMergeRequest(ctx, 1, 2)

	require.NoError(t, err)
	assert.Equal(t, expectedMR, mr)
	repo.AssertExpectations(t)
}

func TestApp_UpdateMergeRequest(t *testing.T) {
	ctx := context.Background()
	repo := &mocks.MockRepository{}
	app := &App{repo: repo, teamUsers: []string{}}

	assigneeID := 1
	reviewerIDs := []int{2, 3}

	repo.On("UpdateMergeRequest", ctx, 1, 2, &assigneeID, reviewerIDs).Return(nil)

	err := app.UpdateMergeRequest(ctx, 1, 2, &assigneeID, reviewerIDs)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestApp_AnalyzeWorkload(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		teamUsers []string
		setupMock func(*mocks.MockRepository)
		validate  func(*testing.T, []*domain.UserWorkload, error)
	}{
		{
			name:      "successful analysis",
			teamUsers: []string{"user1", "user2"},
			setupMock: func(m *mocks.MockRepository) {
				users := []*domain.User{
					{ID: 1, Username: "user1", Email: "user1@example.com"},
					{ID: 2, Username: "user2", Email: "user2@example.com"},
				}
				m.On("GetAllUsers", ctx).Return(users, nil)
				m.On("ListCommits", ctx, 1).Return([]*domain.Commit{
					{AuthorEmail: "user1@example.com"},
					{AuthorEmail: "user1@example.com"},
				}, nil)
				m.On("ListMergeRequests", ctx, "opened", []string{"all"}).Return([]*domain.MergeRequest{
					{
						ID: 1, IID: 1, ProjectID: 1,
						Assignee: &domain.User{ID: 1},
						Author:   &domain.User{ID: 3},
					},
				}, nil)
				m.On("GetUserByUsername", ctx, "user1").Return(users[0], nil)
				m.On("GetUserByUsername", ctx, "user2").Return(users[1], nil)
				m.On("GetMergeRequestApprovals", mock.Anything, 1, 1).Return([]*domain.User{}, nil)
			},
			validate: func(t *testing.T, workloads []*domain.UserWorkload, err error) {
				require.NoError(t, err)
				require.Len(t, workloads, 2)
				assert.Equal(t, 1, workloads[0].User.ID)
				assert.Equal(t, 2, workloads[0].Commits)
				assert.Equal(t, 1, workloads[0].MRCount)
			},
		},
		{
			name:      "repository error on getAllUsers",
			teamUsers: []string{"user1"},
			setupMock: func(m *mocks.MockRepository) {
				m.On("GetAllUsers", ctx).Return(nil, errors.New("db error"))
			},
			validate: func(t *testing.T, workloads []*domain.UserWorkload, err error) {
				require.Error(t, err)
				assert.Nil(t, workloads)
			},
		},
		{
			name:      "user not found in team",
			teamUsers: []string{"user1", "nonexistent"},
			setupMock: func(m *mocks.MockRepository) {
				users := []*domain.User{
					{ID: 1, Username: "user1", Email: "user1@example.com"},
				}
				m.On("GetAllUsers", ctx).Return(users, nil)
				m.On("ListCommits", ctx, 1).Return([]*domain.Commit{}, nil)
				m.On("ListMergeRequests", ctx, "opened", []string{"all"}).Return([]*domain.MergeRequest{}, nil)
				m.On("GetUserByUsername", ctx, "user1").Return(users[0], nil)
				m.On("GetUserByUsername", ctx, "nonexistent").Return(nil, errors.New("not found"))
			},
			validate: func(t *testing.T, workloads []*domain.UserWorkload, err error) {
				require.NoError(t, err)
				require.Len(t, workloads, 1) // Only user1 found
			},
		},
		{
			name:      "error listing commits",
			teamUsers: []string{"user1"},
			setupMock: func(m *mocks.MockRepository) {
				users := []*domain.User{
					{ID: 1, Username: "user1", Email: "user1@example.com"},
				}
				m.On("GetAllUsers", ctx).Return(users, nil)
				m.On("ListCommits", ctx, 1).Return(nil, errors.New("commit error"))
			},
			validate: func(t *testing.T, workloads []*domain.UserWorkload, err error) {
				require.Error(t, err)
				assert.Nil(t, workloads)
			},
		},
		{
			name:      "error listing merge requests",
			teamUsers: []string{"user1"},
			setupMock: func(m *mocks.MockRepository) {
				users := []*domain.User{
					{ID: 1, Username: "user1", Email: "user1@example.com"},
				}
				m.On("GetAllUsers", ctx).Return(users, nil)
				m.On("ListCommits", ctx, 1).Return([]*domain.Commit{}, nil)
				m.On("ListMergeRequests", ctx, "opened", []string{"all"}).Return(nil, errors.New("mr error"))
			},
			validate: func(t *testing.T, workloads []*domain.UserWorkload, err error) {
				require.Error(t, err)
				assert.Nil(t, workloads)
			},
		},
		{
			name:      "user already approved MR",
			teamUsers: []string{"user1"},
			setupMock: func(m *mocks.MockRepository) {
				users := []*domain.User{
					{ID: 1, Username: "user1", Email: "user1@example.com"},
				}
				m.On("GetAllUsers", ctx).Return(users, nil)
				m.On("ListCommits", ctx, 1).Return([]*domain.Commit{}, nil)
				m.On("ListMergeRequests", ctx, "opened", []string{"all"}).Return([]*domain.MergeRequest{
					{
						ID: 1, IID: 1, ProjectID: 1,
						Assignee: &domain.User{ID: 1},
						Author:   &domain.User{ID: 2},
					},
				}, nil)
				m.On("GetUserByUsername", ctx, "user1").Return(users[0], nil)
				m.On("GetMergeRequestApprovals", mock.Anything, 1, 1).Return([]*domain.User{{ID: 1}}, nil)
			},
			validate: func(t *testing.T, workloads []*domain.UserWorkload, err error) {
				require.NoError(t, err)
				require.Len(t, workloads, 1)
				assert.Equal(t, 0, workloads[0].MRCount) // Already approved
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mocks.MockRepository{}
			tt.setupMock(repo)
			app := &App{repo: repo, teamUsers: tt.teamUsers}

			workloads, err := app.AnalyzeWorkload(ctx, 1)

			tt.validate(t, workloads, err)
			repo.AssertExpectations(t)
		})
	}
}

func TestApp_AnalyzeActiveMRs(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		teamUsers []string
		setupMock func(*mocks.MockRepository)
		validate  func(*testing.T, []*domain.UserWorkload, error)
	}{
		{
			name:      "successful analysis",
			teamUsers: []string{"user1"},
			setupMock: func(m *mocks.MockRepository) {
				user := &domain.User{ID: 1, Username: "user1"}
				mrs := []*domain.MergeRequest{
					{
						ID: 1, IID: 1, ProjectID: 1,
						Assignee: user,
						Author:   &domain.User{ID: 2},
					},
				}
				m.On("ListMergeRequests", mock.Anything, "opened", []string{"all"}).Return(mrs, nil)
				m.On("GetUserByUsername", mock.Anything, "user1").Return(user, nil)
				m.On("GetMergeRequestApprovals", mock.Anything, 1, 1).Return([]*domain.User{}, nil)
			},
			validate: func(t *testing.T, workloads []*domain.UserWorkload, err error) {
				require.NoError(t, err)
				require.Len(t, workloads, 1)
				assert.Equal(t, 1, workloads[0].MRCount)
				assert.Len(t, workloads[0].ActiveMRs, 1)
			},
		},
		{
			name:      "error listing merge requests",
			teamUsers: []string{"user1"},
			setupMock: func(m *mocks.MockRepository) {
				m.On("ListMergeRequests", mock.Anything, "opened", []string{"all"}).Return(nil, errors.New("api error"))
			},
			validate: func(t *testing.T, workloads []*domain.UserWorkload, err error) {
				require.Error(t, err)
				assert.Nil(t, workloads)
			},
		},
		{
			name:      "user not found",
			teamUsers: []string{"user1"},
			setupMock: func(m *mocks.MockRepository) {
				m.On("ListMergeRequests", mock.Anything, "opened", []string{"all"}).Return([]*domain.MergeRequest{}, nil)
				m.On("GetUserByUsername", mock.Anything, "user1").Return(nil, errors.New("not found"))
			},
			validate: func(t *testing.T, workloads []*domain.UserWorkload, err error) {
				require.NoError(t, err)
				assert.Empty(t, workloads) // User not found, skipped
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mocks.MockRepository{}
			tt.setupMock(repo)
			app := &App{repo: repo, teamUsers: tt.teamUsers}

			workloads, err := app.AnalyzeActiveMRs(ctx)

			tt.validate(t, workloads, err)
			repo.AssertExpectations(t)
		})
	}
}

func TestApp_SuggestAssigneeAndReviewer(t *testing.T) {
	ctx := context.Background()
	app := &App{teamUsers: []string{}}

	tests := []struct {
		name      string
		mr        *domain.MergeRequest
		workloads []*domain.UserWorkload
		validate  func(*testing.T, *domain.User, *domain.User, error)
	}{
		{
			name: "successful suggestion",
			mr: &domain.MergeRequest{
				Author: &domain.User{ID: 1},
			},
			workloads: []*domain.UserWorkload{
				{
					User:    &domain.User{ID: 2, Status: domain.UserStatus{Availability: "available"}},
					MRCount: 1,
					Commits: 10,
				},
				{
					User:    &domain.User{ID: 3, Status: domain.UserStatus{Availability: "available"}},
					MRCount: 2,
					Commits: 5,
				},
			},
			validate: func(t *testing.T, assignee, reviewer *domain.User, err error) {
				require.NoError(t, err)
				assert.NotNil(t, assignee)
				assert.NotNil(t, reviewer)
				assert.NotEqual(t, assignee.ID, reviewer.ID)
				assert.NotEqual(t, 1, assignee.ID) // Not the author
			},
		},
		{
			name:      "no team members",
			mr:        &domain.MergeRequest{},
			workloads: []*domain.UserWorkload{},
			validate: func(t *testing.T, assignee, reviewer *domain.User, err error) {
				require.Error(t, err)
				assert.Nil(t, assignee)
				assert.Nil(t, reviewer)
			},
		},
		{
			name: "no available team members",
			mr:   &domain.MergeRequest{},
			workloads: []*domain.UserWorkload{
				{
					User:    &domain.User{ID: 1, Status: domain.UserStatus{Availability: "busy"}},
					MRCount: 1,
				},
			},
			validate: func(t *testing.T, assignee, reviewer *domain.User, err error) {
				require.Error(t, err)
				assert.Nil(t, assignee)
				assert.Nil(t, reviewer)
			},
		},
		{
			name: "single available user",
			mr: &domain.MergeRequest{
				Author: &domain.User{ID: 1},
			},
			workloads: []*domain.UserWorkload{
				{
					User:    &domain.User{ID: 2, Status: domain.UserStatus{Availability: "available"}},
					MRCount: 1,
					Commits: 10,
				},
			},
			validate: func(t *testing.T, assignee, reviewer *domain.User, err error) {
				// Should work with single user, but reviewer might be nil if assignee is same
				if err == nil {
					assert.NotNil(t, assignee)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assignee, reviewer, err := app.SuggestAssigneeAndReviewer(ctx, tt.mr, tt.workloads)
			tt.validate(t, assignee, reviewer, err)
		})
	}
}

func TestApp_SortMergeRequestsByPriority(t *testing.T) {
	app := &App{teamUsers: []string{}}

	now := time.Now()
	mrs := []*domain.MergeRequestWithStatus{
		{
			MergeRequest:     &domain.MergeRequest{ID: 1, UpdatedAt: now.Add(-1 * time.Hour)},
			IsCurrentBranch:  false,
			IsCurrentProject: false,
		},
		{
			MergeRequest:     &domain.MergeRequest{ID: 2, UpdatedAt: now.Add(-2 * time.Hour)},
			IsCurrentBranch:  true,
			IsCurrentProject: true,
		},
		{
			MergeRequest:     &domain.MergeRequest{ID: 3, UpdatedAt: now},
			IsCurrentBranch:  false,
			IsCurrentProject: true,
		},
	}

	sorted := app.SortMergeRequestsByPriority(mrs, 1, "main")

	require.Len(t, sorted, 3)
	// Current branch should be first
	assert.True(t, sorted[0].IsCurrentBranch)
	// Current project should be second
	assert.True(t, sorted[1].IsCurrentProject && !sorted[1].IsCurrentBranch)
}

func TestApp_GetMergeRequestsWithStatus(t *testing.T) {
	ctx := context.Background()
	repo := &mocks.MockRepository{}
	app := &App{repo: repo, teamUsers: []string{}}

	now := time.Now()
	// Use 7 calendar days ago to ensure it's definitely more than 3 working days ago
	// regardless of what day of the week the test runs
	mrs := []*domain.MergeRequest{
		{
			ID: 1, IID: 1, ProjectID: 1,
			UpdatedAt:    now.Add(-7 * 24 * time.Hour), // 7 days ago (definitely > 3 working days)
			SourceBranch: "main",
		},
	}

	// ListMergeRequests is called with "opened" and no scope (empty variadic)
	repo.On("ListMergeRequests", mock.Anything, "opened", []string{}).Return(mrs, nil)
	repo.On("GetMergeRequestApprovals", mock.Anything, 1, 1).Return([]*domain.User{}, nil)

	// GetCurrentProjectInfo may succeed if we're in a git repo, so mock GetProject
	// It will try to parse the git remote URL and call GetProject
	// We'll mock it to return an error so it's handled gracefully
	repo.On("GetProject", mock.Anything, mock.Anything).Return(nil, assert.AnError).Maybe()

	mrsWithStatus, err := app.GetMergeRequestsWithStatus(ctx)

	// Should work regardless of whether GetCurrentProjectInfo succeeds or fails
	require.NoError(t, err)
	require.Len(t, mrsWithStatus, 1)
	assert.True(t, mrsWithStatus[0].IsStalled) // Updated 7 days ago (definitely > 3 working days)
}

func TestApp_GetMergeRequestApprovals(t *testing.T) {
	ctx := context.Background()
	repo := &mocks.MockRepository{}
	app := &App{repo: repo, teamUsers: []string{}}

	approvals := []*domain.User{
		{ID: 1, Username: "reviewer1"},
		{ID: 2, Username: "reviewer2"},
	}

	repo.On("GetMergeRequestApprovals", ctx, 1, 2).Return(approvals, nil)

	result, err := app.GetMergeRequestApprovals(ctx, 1, 2)

	require.NoError(t, err)
	assert.Equal(t, approvals, result)
	repo.AssertExpectations(t)
}

func TestApp_GetMyReviewWorkloadWithStatus(t *testing.T) {
	ctx := context.Background()
	repo := &mocks.MockRepository{}
	app := &App{repo: repo, teamUsers: []string{}}

	currentUser := &domain.User{ID: 1, Username: "current"}
	now := time.Now()
	mrs := []*domain.MergeRequest{
		{
			ID: 1, IID: 1, ProjectID: 1,
			Assignee:  currentUser,
			Author:    &domain.User{ID: 2},
			Draft:     false,
			UpdatedAt: now.Add(-1 * time.Hour),
		},
		{
			ID: 2, IID: 2, ProjectID: 1,
			Reviewers: []*domain.User{currentUser},
			Author:    &domain.User{ID: 3},
			Draft:     false,
			UpdatedAt: now.Add(-2 * time.Hour),
		},
		{
			ID: 3, IID: 3, ProjectID: 1,
			Assignee: currentUser,
			Author:   currentUser, // User is author, should be skipped
			Draft:    false,
		},
		{
			ID: 4, IID: 4, ProjectID: 1,
			Assignee: currentUser,
			Author:   &domain.User{ID: 4},
			Draft:    true, // Draft, should be skipped
		},
		{
			ID: 5, IID: 5, ProjectID: 1,
			Assignee: currentUser,
			Author:   &domain.User{ID: 5},
			Draft:    false,
		},
	}

	repo.On("GetCurrentUser", mock.Anything).Return(currentUser, nil)
	repo.On("ListMergeRequests", mock.Anything, "opened", []string{"all"}).Return(mrs, nil)
	repo.On("GetMergeRequestApprovals", mock.Anything, 1, 1).Return([]*domain.User{}, nil)
	repo.On("GetMergeRequestApprovals", mock.Anything, 1, 2).Return([]*domain.User{}, nil)
	repo.On("GetMergeRequestApprovals", mock.Anything, 1, 5).Return([]*domain.User{{ID: 1}}, nil) // Already approved
	repo.On("GetProject", mock.Anything, mock.Anything).Return(nil, assert.AnError).Maybe()

	mrsWithStatus, err := app.GetMyReviewWorkloadWithStatus(ctx)

	require.NoError(t, err)
	require.Len(t, mrsWithStatus, 2) // Should exclude draft, author's own MRs, and already approved
}

func TestApp_buildEmailToUserIDMap(t *testing.T) {
	ctx := context.Background()
	repo := &mocks.MockRepository{}
	app := &App{repo: repo, teamUsers: []string{}}

	users := []*domain.User{
		{ID: 1, Email: "user1@example.com"},
		{ID: 2, Email: "user2@example.com"},
		{ID: 3, Email: ""}, // No email, should be skipped
	}

	repo.On("GetAllUsers", ctx).Return(users, nil)

	emailMap, err := app.buildEmailToUserIDMap(ctx)

	require.NoError(t, err)
	assert.Equal(t, 1, emailMap["user1@example.com"])
	assert.Equal(t, 2, emailMap["user2@example.com"])
	assert.NotContains(t, emailMap, "")
	repo.AssertExpectations(t)
}

func TestApp_countUserCommits(t *testing.T) {
	ctx := context.Background()
	repo := &mocks.MockRepository{}
	app := &App{repo: repo, teamUsers: []string{}}

	emailToUserID := map[string]int{
		"user1@example.com": 1,
	}

	commits := []*domain.Commit{
		{AuthorEmail: "user1@example.com"},
		{AuthorEmail: "user1@example.com"},
		{AuthorEmail: "user2@example.com"}, // Not in map, will try to fetch
		{AuthorEmail: ""},                  // Empty email, should be skipped
	}

	repo.On("ListCommits", ctx, 1).Return(commits, nil)
	repo.On("GetUserByUsername", ctx, "user2").Return(&domain.User{ID: 2, Email: "user2@example.com"}, nil)

	userCommits, err := app.countUserCommits(ctx, 1, emailToUserID)

	require.NoError(t, err)
	assert.Equal(t, 2, userCommits[1])
	assert.Equal(t, 1, userCommits[2])
	repo.AssertExpectations(t)
}

func TestApp_countUserCommits_UserNotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mocks.MockRepository{}
	app := &App{repo: repo, teamUsers: []string{}}

	emailToUserID := map[string]int{}

	commits := []*domain.Commit{
		{AuthorEmail: "unknown@example.com"},
	}

	repo.On("ListCommits", ctx, 1).Return(commits, nil)
	repo.On("GetUserByUsername", ctx, "unknown").Return(nil, errors.New("not found"))

	userCommits, err := app.countUserCommits(ctx, 1, emailToUserID)

	require.NoError(t, err) // Error is ignored, commit is skipped
	assert.Empty(t, userCommits)
	repo.AssertExpectations(t)
}

func TestHelperFunctions(t *testing.T) {
	t.Run("isUserInvolvedInMR", func(t *testing.T) {
		userID := 1
		tests := []struct {
			name     string
			mr       *domain.MergeRequest
			expected bool
		}{
			{
				name:     "nil MR",
				mr:       nil,
				expected: false,
			},
			{
				name: "draft MR",
				mr: &domain.MergeRequest{
					Draft:    true,
					Assignee: &domain.User{ID: userID},
				},
				expected: false,
			},
			{
				name: "user is author",
				mr: &domain.MergeRequest{
					Author: &domain.User{ID: userID},
				},
				expected: false,
			},
			{
				name: "user is assignee",
				mr: &domain.MergeRequest{
					Assignee: &domain.User{ID: userID},
					Author:   &domain.User{ID: 2},
				},
				expected: true,
			},
			{
				name: "user is reviewer",
				mr: &domain.MergeRequest{
					Reviewers: []*domain.User{{ID: userID}},
					Author:    &domain.User{ID: 2},
				},
				expected: true,
			},
			{
				name: "user is both assignee and reviewer",
				mr: &domain.MergeRequest{
					Assignee:  &domain.User{ID: userID},
					Reviewers: []*domain.User{{ID: userID}},
					Author:    &domain.User{ID: 2},
				},
				expected: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := isUserInvolvedInMR(tt.mr, userID)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("hasUserApprovedMR", func(t *testing.T) {
		userID := 1
		approvals := []*domain.User{
			{ID: 1},
			{ID: 2},
		}

		assert.True(t, hasUserApprovedMR(approvals, userID))
		assert.False(t, hasUserApprovedMR(approvals, 3))
		assert.False(t, hasUserApprovedMR(nil, userID))
		assert.False(t, hasUserApprovedMR([]*domain.User{}, userID))
	})

	t.Run("isUserAvailable", func(t *testing.T) {
		tests := []struct {
			name     string
			user     *domain.User
			expected bool
		}{
			{
				name: "available user",
				user: &domain.User{
					Status: domain.UserStatus{
						Message:      "Working",
						Availability: "available",
					},
				},
				expected: true,
			},
			{
				name: "busy user",
				user: &domain.User{
					Status: domain.UserStatus{
						Availability: "busy",
					},
				},
				expected: false,
			},
			{
				name: "ooo user",
				user: &domain.User{
					Status: domain.UserStatus{
						Message: "OOO",
					},
				},
				expected: false,
			},
			{
				name: "vacation user",
				user: &domain.User{
					Status: domain.UserStatus{
						Message: "On vacation",
					},
				},
				expected: false,
			},
			{
				name: "case insensitive ooo",
				user: &domain.User{
					Status: domain.UserStatus{
						Message: "ooo until next week",
					},
				},
				expected: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := isUserAvailable(tt.user)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("calculateAssigneeScore", func(t *testing.T) {
		score := calculateAssigneeScore(10, 2)
		assert.InDelta(t, 10.0/3.0, score, 0.0001)

		score = calculateAssigneeScore(0, 0)
		assert.InDelta(t, 0.0, score, 0.0001)

		score = calculateAssigneeScore(5, 0)
		assert.InDelta(t, 5.0, score, 0.0001)
	})

	t.Run("subtractWorkingDays", func(t *testing.T) {
		// Monday
		date := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC) // Monday, Jan 8, 2024
		result := subtractWorkingDays(date, 3)
		// Should be previous Wednesday (3 working days back: Mon->Fri->Thu->Wed)
		expected := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC) // Wednesday, Jan 3, 2024
		assert.Equal(t, expected.Weekday(), result.Weekday())
		assert.True(t, result.Before(date))

		// Test weekend handling - start on Monday, go back 1 day should be Friday
		date = time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC) // Monday
		result = subtractWorkingDays(date, 1)
		assert.Equal(t, time.Friday, result.Weekday())

		// Test starting on Saturday
		date = time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC) // Saturday
		result = subtractWorkingDays(date, 1)
		// Should skip Saturday and go to Friday
		assert.Equal(t, time.Friday, result.Weekday())

		// Test starting on Sunday
		date = time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC) // Sunday
		result = subtractWorkingDays(date, 1)
		// Should skip Sunday and go to Friday
		assert.Equal(t, time.Friday, result.Weekday())
	})
}

func TestApp_fetchMRApprovals(t *testing.T) {
	ctx := context.Background()
	repo := &mocks.MockRepository{}
	app := &App{repo: repo, teamUsers: []string{}}

	mrs := []*domain.MergeRequest{
		{ID: 1, IID: 1, ProjectID: 1},
		{ID: 2, IID: 2, ProjectID: 1},
	}

	repo.On("GetMergeRequestApprovals", mock.Anything, 1, 1).Return([]*domain.User{{ID: 1}}, nil)
	repo.On("GetMergeRequestApprovals", mock.Anything, 1, 2).Return([]*domain.User{{ID: 2}}, nil)

	approvalsMap, err := app.fetchMRApprovals(ctx, mrs)

	require.NoError(t, err)
	require.Len(t, approvalsMap, 2)
	assert.Len(t, approvalsMap[1], 1)
	assert.Len(t, approvalsMap[2], 1)
	repo.AssertExpectations(t)
}

func TestApp_fetchMRApprovals_Error(t *testing.T) {
	ctx := context.Background()
	repo := &mocks.MockRepository{}
	app := &App{repo: repo, teamUsers: []string{}}

	mrs := []*domain.MergeRequest{
		{ID: 1, IID: 1, ProjectID: 1},
	}

	repo.On("GetMergeRequestApprovals", mock.Anything, 1, 1).Return(nil, errors.New("api error"))

	approvalsMap, err := app.fetchMRApprovals(ctx, mrs)

	require.Error(t, err)
	assert.Nil(t, approvalsMap)
	repo.AssertExpectations(t)
}

func TestApp_fetchMRApprovals_EmptyList(t *testing.T) {
	ctx := context.Background()
	repo := &mocks.MockRepository{}
	app := &App{repo: repo, teamUsers: []string{}}

	approvalsMap, err := app.fetchMRApprovals(ctx, []*domain.MergeRequest{})

	require.NoError(t, err)
	assert.NotNil(t, approvalsMap)
	assert.Empty(t, approvalsMap)
}
