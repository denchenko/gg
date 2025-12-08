package ascii

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/denchenko/gg/internal/core/domain"
)

const (
	unknownProject    = "Unknown Project"
	noneString        = "None"
	urlPartsCount     = 2
	descriptionMaxLen = 100
	descriptionTrunc  = 97
	boxWidth          = 100
	boxTitlePadding   = 5
	boxBottomPadding  = 2
	minApprovalCount  = 2
)

var (
	//go:embed team_review.tmpl
	teamReviewTemplate string

	//go:embed mr_roulette.tmpl
	mrRouletteTemplate string

	//go:embed my_mr.tmpl
	myMRTemplate string

	//go:embed my_review.tmpl
	myReviewTemplate string
)

// TeamWorkloadData holds data for team workload templates.
type TeamWorkloadData struct {
	Workloads []*domain.UserWorkload
	Timestamp time.Time
}

// MergeRequestWithApprovals represents a merge request with its approvals.
type MergeRequestWithApprovals struct {
	*domain.MergeRequest
	Approvals []*domain.User
}

// MergeRequestWithStatus represents a merge request with status information for templates.
type MergeRequestWithStatus struct {
	*domain.MergeRequest
	Approvals        []*domain.User
	ApprovalCount    int
	IsStalled        bool
	IsCurrentBranch  bool
	IsCurrentProject bool
}

// MergeRequestStatusData holds data for merge request status templates.
type MergeRequestStatusData struct {
	MergeRequests []*MergeRequestWithApprovals
	Timestamp     time.Time
}

// MyMergeRequestStatusData holds data for my merge request status templates.
type MyMergeRequestStatusData struct {
	MergeRequests     []*domain.MergeRequestWithStatus
	OtherMRsByProject map[string][]*domain.MergeRequestWithStatus
	Timestamp         time.Time
}

// MyReviewWorkloadData holds data for my review workload templates.
type MyReviewWorkloadData struct {
	MRsByProject map[string][]*domain.MergeRequestWithStatus
	Timestamp    time.Time
}

// MRRouletteData holds data for MR roulette templates.
type MRRouletteData struct {
	MergeRequest      *domain.MergeRequest
	MRURL             string
	Workloads         []*domain.UserWorkload
	SuggestedAssignee *domain.User
	SuggestedReviewer *domain.User
	Timestamp         time.Time
}

// FormatTeamWorkload formats team workload data using a template.
func FormatTeamWorkload(workloads []*domain.UserWorkload) (string, error) {
	return executeWorkloadTemplate(teamReviewTemplate, workloads)
}

// FormatMyMergeRequestStatus formats my merge request status data using a template.
func FormatMyMergeRequestStatus(mrs []*domain.MergeRequestWithStatus) (string, error) {
	otherMRsByProject := make(map[string][]*domain.MergeRequestWithStatus)
	getProjectName := func(webURL string) string {
		parts := strings.Split(webURL, "/-/merge_requests/")
		if len(parts) != urlPartsCount {
			return unknownProject
		}
		projectPart := parts[0]
		if strings.Contains(projectPart, "gitlab.com/") {
			projectPart = strings.Split(projectPart, "gitlab.com/")[1]
		} else if strings.Contains(projectPart, "gitlab.twinby.tech/") {
			projectPart = strings.Split(projectPart, "gitlab.twinby.tech/")[1]
		}

		return projectPart
	}
	for _, mr := range mrs {
		if !mr.IsCurrentProject {
			projectName := getProjectName(mr.WebURL)
			otherMRsByProject[projectName] = append(otherMRsByProject[projectName], mr)
		}
	}

	return executeMyStatusTemplate(myMRTemplate, mrs, otherMRsByProject)
}

// FormatMRRoulette formats MR roulette data using a template.
func FormatMRRoulette(
	mr *domain.MergeRequest,
	mrURL string,
	workloads []*domain.UserWorkload,
	suggestedAssignee, suggestedReviewer *domain.User,
) (string, error) {
	return executeMRRouletteTemplate(mrRouletteTemplate, mr, mrURL, workloads, suggestedAssignee, suggestedReviewer)
}

// FormatMyReviewWorkload formats my review workload data using a template.
func FormatMyReviewWorkload(mrs []*domain.MergeRequestWithStatus) (string, error) {
	mrsByProject := make(map[string][]*domain.MergeRequestWithStatus)
	getProjectName := func(webURL string) string {
		parts := strings.Split(webURL, "/-/merge_requests/")
		if len(parts) != urlPartsCount {
			return unknownProject
		}
		projectPart := parts[0]
		if strings.Contains(projectPart, "gitlab.com/") {
			projectPart = strings.Split(projectPart, "gitlab.com/")[1]
		} else if strings.Contains(projectPart, "gitlab.twinby.tech/") {
			projectPart = strings.Split(projectPart, "gitlab.twinby.tech/")[1]
		}

		return projectPart
	}
	for _, mr := range mrs {
		projectName := getProjectName(mr.WebURL)
		mrsByProject[projectName] = append(mrsByProject[projectName], mr)
	}

	return executeMyReviewTemplate(myReviewTemplate, mrsByProject)
}

func executeWorkloadTemplate(templateStr string, workloads []*domain.UserWorkload) (string, error) {
	tmpl, err := template.New("teamWorkload").Funcs(getWorkloadTemplateFuncs()).Parse(templateStr)

	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	data := TeamWorkloadData{
		Workloads: workloads,
		Timestamp: time.Now(),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

func getWorkloadTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"getRole": func(mr *domain.MergeRequest, user *domain.User) string {
			if mr.Assignee != nil && mr.Assignee.ID == user.ID {
				return "Assignee"
			}

			return "Reviewer"
		},
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"truncateDescription": truncateDescription,
		"sub": func(a, b int) int {
			return a - b
		},
		"len": func(slice []*domain.MergeRequest) int {
			return len(slice)
		},
		"ne": func(a, b int) bool {
			return a != b
		},
		"formatBoxTitle":  formatBoxTitle,
		"formatBoxBottom": formatBoxBottom,
		"bold": func(text string) string {
			return "\033[1m" + text + "\033[0m"
		},
		"repeat": strings.Repeat,
	}
}

func truncateDescription(desc string) string {
	// Replace multiple consecutive newlines with a single semicolon
	for strings.Contains(desc, "\n\n") {
		desc = strings.ReplaceAll(desc, "\n\n", "; ")
	}
	// Replace remaining single newlines with semicolons
	desc = strings.ReplaceAll(desc, "\n", "; ")
	if len(desc) > descriptionMaxLen {
		return desc[:descriptionTrunc] + "..."
	}

	return desc
}

func formatBoxTitle(title string) string {
	titleMax := boxWidth - boxTitlePadding // space for ┌─, ─┐, and spaces

	// Strip ANSI escape codes for length calculation
	cleanTitle := title
	// Remove \033[1m and \033[0m escape codes
	cleanTitle = strings.ReplaceAll(cleanTitle, "\033[1m", "")
	cleanTitle = strings.ReplaceAll(cleanTitle, "\033[0m", "")

	t := cleanTitle
	if len(t) > titleMax {
		t = t[:titleMax]
	}
	dashCount := boxWidth - len(t) - boxTitlePadding
	if dashCount < 0 {
		dashCount = 0
	}

	return "┌─ " + title + " " + strings.Repeat("─", dashCount) + "┐"
}

func formatBoxBottom() string {
	return "└" + strings.Repeat("─", boxWidth-boxBottomPadding) + "┘"
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func joinUsernames(users []*domain.User) string {
	if len(users) == 0 {
		return noneString
	}
	usernames := make([]string, len(users))
	for i, user := range users {
		usernames[i] = user.Username
	}

	return strings.Join(usernames, ", ")
}

func getProjectName(webURL string) string {
	parts := strings.Split(webURL, "/-/merge_requests/")
	if len(parts) != urlPartsCount {
		return unknownProject
	}

	projectPart := parts[0]
	if strings.Contains(projectPart, "gitlab.com/") {
		projectPart = strings.Split(projectPart, "gitlab.com/")[1]
	} else if strings.Contains(projectPart, "gitlab.twinby.tech/") {
		projectPart = strings.Split(projectPart, "gitlab.twinby.tech/")[1]
	}

	return projectPart
}

func getStatusEmoji(mr *domain.MergeRequestWithStatus) string {
	if mr.IsStalled {
		return "\033[31m[stalled]\033[0m "
	}
	if mr.ApprovalCount >= minApprovalCount {
		return "\033[32m[ready-to-merge]\033[0m "
	}

	return ""
}

func getMRRouletteTemplateFuncs(workloads []*domain.UserWorkload) template.FuncMap {
	return template.FuncMap{
		"formatTime":    formatTime,
		"joinUsernames": joinUsernames,
		"getWorkloadMRCount": func(userID int) int {
			for _, w := range workloads {
				if w.User.ID == userID {
					return w.MRCount
				}
			}

			return 0
		},
		"getWorkloadCommits": func(userID int) int {
			for _, w := range workloads {
				if w.User.ID == userID {
					return w.Commits
				}
			}

			return 0
		},
	}
}

func getMyStatusTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatTime":      formatTime,
		"joinUsernames":   joinUsernames,
		"getStatusEmoji":  getStatusEmoji,
		"getProjectName":  getProjectName,
		"repeat":          strings.Repeat,
		"formatBoxTitle":  formatBoxTitle,
		"formatBoxBottom": formatBoxBottom,
		"bold": func(text string) string {
			return "\033[1m" + text + "\033[0m"
		},
		"add": func(a, b int) int {
			return a + b
		},
		"gte": func(a, b int) bool {
			return a >= b
		},
	}
}

func getMyReviewTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatTime":          formatTime,
		"joinUsernames":       joinUsernames,
		"getStatusEmoji":      getStatusEmoji,
		"getProjectName":      getProjectName,
		"truncateDescription": truncateDescription,
		"formatBoxTitle":      formatBoxTitle,
		"formatBoxBottom":     formatBoxBottom,
		"bold": func(text string) string {
			return "\033[1m" + text + "\033[0m"
		},
		"gte": func(a, b int) bool {
			return a >= b
		},
	}
}

func executeMRRouletteTemplate(
	templateStr string,
	mr *domain.MergeRequest,
	mrURL string,
	workloads []*domain.UserWorkload,
	suggestedAssignee, suggestedReviewer *domain.User,
) (string, error) {
	tmpl, err := template.New("mrRoulette").Funcs(getMRRouletteTemplateFuncs(workloads)).Parse(templateStr)

	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	data := MRRouletteData{
		MergeRequest:      mr,
		MRURL:             mrURL,
		Workloads:         workloads,
		SuggestedAssignee: suggestedAssignee,
		SuggestedReviewer: suggestedReviewer,
		Timestamp:         time.Now(),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

func executeMyStatusTemplate(
	templateStr string,
	mrs []*domain.MergeRequestWithStatus,
	otherMRsByProject map[string][]*domain.MergeRequestWithStatus,
) (string, error) {
	tmpl, err := template.New("myMergeRequestStatus").Funcs(getMyStatusTemplateFuncs()).Parse(templateStr)

	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	data := MyMergeRequestStatusData{
		MergeRequests:     mrs,
		OtherMRsByProject: otherMRsByProject,
		Timestamp:         time.Now(),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

func executeMyReviewTemplate(
	templateStr string,
	mrsByProject map[string][]*domain.MergeRequestWithStatus,
) (string, error) {
	tmpl, err := template.New("myReview").Funcs(getMyReviewTemplateFuncs()).Parse(templateStr)

	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Prepare data
	data := MyReviewWorkloadData{
		MRsByProject: mrsByProject,
		Timestamp:    time.Now(),
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
