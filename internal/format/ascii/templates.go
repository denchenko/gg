package ascii

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/denchenko/gg/internal/core/domain"
	"github.com/denchenko/gg/internal/issue"
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

	// Activity description constants.
	commitTitleMaxLen = 60
	commitTitleTrunc  = 57
	noteBodyMaxLen    = 80
	noteBodyTrunc     = 77
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

	//go:embed my_activity.tmpl
	myActivityTemplate string

	//go:embed mr_status.tmpl
	mrStatusTemplate string
)

// Formatter handles formatting of various data structures using templates.
type Formatter struct {
	issuer *issue.Issuer
}

// NewFormatter creates a new Formatter instance with the given Issuer.
func NewFormatter(issuer *issue.Issuer) *Formatter {
	return &Formatter{
		issuer: issuer,
	}
}

// TeamWorkloadData holds data for team workload templates.
type TeamWorkloadData struct {
	Workloads []*domain.UserWorkload
	Timestamp time.Time
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

// MyActivityData holds data for my activity templates.
type MyActivityData struct {
	EventsByProject map[string][]*domain.Event
	Timestamp       time.Time
}

// FormatTeamWorkload formats team workload data using a template.
func (f *Formatter) FormatTeamWorkload(workloads []*domain.UserWorkload) (string, error) {
	return f.executeWorkloadTemplate(teamReviewTemplate, workloads)
}

// FormatMyMergeRequestStatus formats my merge request status data using a template.
func (f *Formatter) FormatMyMergeRequestStatus(baseURL string, mrs []*domain.MergeRequestWithStatus) (string, error) {
	otherMRsByProject := make(map[string][]*domain.MergeRequestWithStatus)
	for _, mr := range mrs {
		if !mr.IsCurrentProject {
			projectName := getProjectName(baseURL, mr.WebURL)
			otherMRsByProject[projectName] = append(otherMRsByProject[projectName], mr)
		}
	}

	return f.executeMyStatusTemplate(baseURL, myMRTemplate, mrs, otherMRsByProject)
}

// FormatMRRoulette formats MR roulette data using a template.
func (f *Formatter) FormatMRRoulette(
	mr *domain.MergeRequest,
	mrURL string,
	workloads []*domain.UserWorkload,
	suggestedAssignee, suggestedReviewer *domain.User,
) (string, error) {
	return f.executeMRRouletteTemplate(mrRouletteTemplate, mr, mrURL, workloads, suggestedAssignee, suggestedReviewer)
}

// FormatMyReviewWorkload formats my review workload data using a template.
func (f *Formatter) FormatMyReviewWorkload(baseURL string, mrs []*domain.MergeRequestWithStatus) (string, error) {
	mrsByProject := make(map[string][]*domain.MergeRequestWithStatus)
	for _, mr := range mrs {
		projectName := getProjectName(baseURL, mr.WebURL)
		mrsByProject[projectName] = append(mrsByProject[projectName], mr)
	}

	return f.executeMyReviewTemplate(baseURL, myReviewTemplate, mrsByProject)
}

// FormatMyActivity formats my activity data using a template.
func (f *Formatter) FormatMyActivity(baseURL string, events []*domain.Event) (string, error) {
	eventsByProject := make(map[string][]*domain.Event)
	for _, event := range events {
		projectName := event.ProjectPath
		if projectName == "" {
			projectName = unknownProject
		}
		eventsByProject[projectName] = append(eventsByProject[projectName], event)
	}

	return f.executeMyActivityTemplate(baseURL, myActivityTemplate, eventsByProject)
}

// FormatMRStatus formats a single merge request status using a template.
func (f *Formatter) FormatMRStatus(baseURL string, mr *domain.MergeRequestWithStatus) (string, error) {
	return f.executeMRStatusTemplate(baseURL, mrStatusTemplate, mr)
}

func (f *Formatter) executeWorkloadTemplate(templateStr string, workloads []*domain.UserWorkload) (string, error) {
	tmpl, err := template.New("teamWorkload").Funcs(f.getWorkloadTemplateFuncs()).Parse(templateStr)

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

func (f *Formatter) getWorkloadTemplateFuncs() template.FuncMap {
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
		"getIssueURL": func(title string) string {
			issueNumber := f.issuer.ExtractNumber(title)
			url, _ := f.issuer.MakeURL(issueNumber)

			return url
		},
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

func getProjectName(baseURL, webURL string) string {
	parts := strings.Split(webURL, "/-/merge_requests/")
	if len(parts) != urlPartsCount {
		return unknownProject
	}

	projectPart := parts[0]
	// Remove the base URL prefix to get the project path
	if strings.HasPrefix(projectPart, baseURL+"/") {
		projectPart = strings.TrimPrefix(projectPart, baseURL+"/")
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

func (f *Formatter) getMRRouletteTemplateFuncs(workloads []*domain.UserWorkload) template.FuncMap {
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
		"getIssueURL": func(title string) string {
			issueNumber := f.issuer.ExtractNumber(title)
			url, _ := f.issuer.MakeURL(issueNumber)

			return url
		},
	}
}

func (f *Formatter) getMyStatusTemplateFuncs(baseURL string) template.FuncMap {
	return template.FuncMap{
		"formatTime":      formatTime,
		"joinUsernames":   joinUsernames,
		"getStatusEmoji":  getStatusEmoji,
		"getProjectName":  func(webURL string) string { return getProjectName(baseURL, webURL) },
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
		"getIssueURL": func(title string) string {
			issueNumber := f.issuer.ExtractNumber(title)
			url, _ := f.issuer.MakeURL(issueNumber)

			return url
		},
	}
}

func (f *Formatter) getMyReviewTemplateFuncs(baseURL string) template.FuncMap {
	return template.FuncMap{
		"formatTime":          formatTime,
		"joinUsernames":       joinUsernames,
		"getStatusEmoji":      getStatusEmoji,
		"getProjectName":      func(webURL string) string { return getProjectName(baseURL, webURL) },
		"truncateDescription": truncateDescription,
		"formatBoxTitle":      formatBoxTitle,
		"formatBoxBottom":     formatBoxBottom,
		"bold": func(text string) string {
			return "\033[1m" + text + "\033[0m"
		},
		"gte": func(a, b int) bool {
			return a >= b
		},
		"getIssueURL": func(title string) string {
			issueNumber := f.issuer.ExtractNumber(title)
			url, _ := f.issuer.MakeURL(issueNumber)

			return url
		},
	}
}

func (f *Formatter) executeMRRouletteTemplate(
	templateStr string,
	mr *domain.MergeRequest,
	mrURL string,
	workloads []*domain.UserWorkload,
	suggestedAssignee, suggestedReviewer *domain.User,
) (string, error) {
	tmpl, err := template.New("mrRoulette").Funcs(f.getMRRouletteTemplateFuncs(workloads)).Parse(templateStr)

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

func (f *Formatter) executeMyStatusTemplate(
	baseURL string,
	templateStr string,
	mrs []*domain.MergeRequestWithStatus,
	otherMRsByProject map[string][]*domain.MergeRequestWithStatus,
) (string, error) {
	tmpl, err := template.New("myMergeRequestStatus").Funcs(f.getMyStatusTemplateFuncs(baseURL)).Parse(templateStr)

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

func (f *Formatter) executeMyReviewTemplate(
	baseURL string,
	templateStr string,
	mrsByProject map[string][]*domain.MergeRequestWithStatus,
) (string, error) {
	tmpl, err := template.New("myReview").Funcs(f.getMyReviewTemplateFuncs(baseURL)).Parse(templateStr)

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

func (f *Formatter) executeMyActivityTemplate(
	baseURL string,
	templateStr string,
	eventsByProject map[string][]*domain.Event,
) (string, error) {
	tmpl, err := template.New("myActivity").Funcs(f.getMyActivityTemplateFuncs(baseURL)).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	data := MyActivityData{
		EventsByProject: eventsByProject,
		Timestamp:       time.Now(),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

func (f *Formatter) executeMRStatusTemplate(
	baseURL string,
	templateStr string,
	mr *domain.MergeRequestWithStatus,
) (string, error) {
	tmpl, err := template.New("mrStatus").Funcs(f.getMyStatusTemplateFuncs(baseURL)).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, mr); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

func (f *Formatter) getMyActivityTemplateFuncs(baseURL string) template.FuncMap {
	return template.FuncMap{
		"formatTime":                formatTime,
		"getProjectName":            func(webURL string) string { return getProjectName(baseURL, webURL) },
		"formatBoxTitle":            formatBoxTitle,
		"formatBoxBottom":           formatBoxBottom,
		"formatActivityDescription": formatActivityDescription,
		"bold": func(text string) string {
			return "\033[1m" + text + "\033[0m"
		},
		"getIssueURL": func(title string) string {
			issueNumber := f.issuer.ExtractNumber(title)
			url, _ := f.issuer.MakeURL(issueNumber)

			return url
		},
	}
}

func formatActivityDescription(event *domain.Event) string {
	action := strings.ToLower(event.Action)
	targetType := strings.ToLower(event.TargetType)

	// Handle push events
	if (targetType == "" || strings.Contains(action, "push") || event.Action == "deleted") && event.PushRef != "" {
		return formatPushEventDescription(event)
	}

	// Handle note/comment events
	if targetType == "note" || strings.Contains(action, "comment") {
		return formatCommentEventDescription(event)
	}

	// Handle merge request events
	if targetType == "mergerequest" || targetType == "merge_request" {
		return formatMergeRequestEventDescription(event)
	}

	// Handle issue events
	if targetType == "issue" {
		return formatIssueEventDescription(event)
	}

	// Default formatter
	return formatDefaultEventDescription(event)
}

func formatPushEventDescription(event *domain.Event) string {
	ref := normalizeRef(event.PushRef)
	refType := getRefType(event.PushRef)
	pushAction := event.PushAction
	if pushAction == "" {
		pushAction = event.Action
	}

	desc := fmt.Sprintf("%s %s %s", pushAction, refType, ref)
	if event.CommitCount > 0 {
		desc += fmt.Sprintf(" (%d commit%s", event.CommitCount, pluralize(event.CommitCount))
		if event.CommitTitle != "" {
			title := truncateText(event.CommitTitle, commitTitleMaxLen, commitTitleTrunc)
			desc += ": " + title
		}
		desc += ")"
	}

	return desc
}

func formatCommentEventDescription(event *domain.Event) string {
	desc := "commented"
	if event.TargetTitle != "" {
		desc += ": " + event.TargetTitle
	}
	if event.NoteBody != "" {
		body := strings.ReplaceAll(event.NoteBody, "\n", " ")
		body = truncateText(body, noteBodyMaxLen, noteBodyTrunc)
		desc += fmt.Sprintf(" (%s)", body)
	}

	return desc
}

func formatMergeRequestEventDescription(event *domain.Event) string {
	desc := event.Action
	if event.TargetTitle != "" {
		desc += ": " + event.TargetTitle
	}

	return desc
}

func formatIssueEventDescription(event *domain.Event) string {
	desc := event.Action
	if event.TargetTitle != "" {
		desc += ": " + event.TargetTitle
	}

	return desc
}

func formatDefaultEventDescription(event *domain.Event) string {
	desc := event.Action
	if event.TargetType != "" {
		desc += " " + event.TargetType
	}
	if event.TargetTitle != "" {
		desc += ": " + event.TargetTitle
	}

	return desc
}

func normalizeRef(ref string) string {
	ref = strings.TrimPrefix(ref, "refs/tags/")
	ref = strings.TrimPrefix(ref, "refs/heads/")

	return ref
}

func getRefType(ref string) string {
	if strings.HasPrefix(ref, "refs/tags/") {
		return "tag"
	}

	return "branch"
}

func truncateText(text string, maxLen, truncLen int) string {
	if len(text) > maxLen {
		return text[:truncLen] + "..."
	}

	return text
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}

	return "s"
}
