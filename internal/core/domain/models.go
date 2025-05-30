package domain

import "time"

type User struct {
	ID       int
	Username string
	Email    string
	Status   UserStatus
}

type UserStatus struct {
	Message      string
	Availability string
}

type MergeRequest struct {
	ID           int
	IID          int
	Title        string
	Description  string
	WebURL       string
	Author       *User
	Assignee     *User
	Reviewers    []*User
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ProjectID    int
	Draft        bool
	SourceBranch string
}

type MergeRequestWithStatus struct {
	*MergeRequest
	Approvals        []*User
	ApprovalCount    int
	IsStalled        bool
	IsCurrentBranch  bool
	IsCurrentProject bool
}

type Commit struct {
	ID          string
	AuthorName  string
	AuthorEmail string
	CreatedAt   time.Time
	Message     string
	WebURL      string
}

type UserWorkload struct {
	User      *User
	MRCount   int
	Commits   int
	ActiveMRs []*MergeRequest
}

type Project struct {
	ID   int
	Path string
}
