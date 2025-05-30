# gg

[![Go Report Card](https://goreportcard.com/badge/github.com/denchenko/gg)](https://goreportcard.com/report/github.com/denchenko/gg)
[![GoDoc](https://godoc.org/github.com/denchenko/gg?status.svg)](https://godoc.org/github.com/denchenko/gg)

A CLI tool for managing GitLab merge requests with intelligent workload analysis and automatic assignment suggestions.

## Configuration

- `GG_TOKEN` (required) - Your GitLab personal access token with `api` scope
- `GG_TEAM` (required) - Comma-separated list of team member usernames (e.g., `user1,user2,user3`)
- `GG_BASE_URL` (optional) - GitLab instance URL (defaults to `https://gitlab.com`)

Example:
```bash
export GG_TOKEN="your-gitlab-token"
export GG_TEAM="alice,bob,charlie"
export GG_BASE_URL="https://gitlab.com"  # Optional, defaults to gitlab.com
```

## Usage

### CLI

**Install:**
```bash
go install github.com/denchenko/gg/cmd/gg@latest
```

The CLI provides commands for managing merge requests and analyzing team workload:

- `gg my mr` - Show your personal merge requests with status information
- `gg my review` - Display your review workload (MRs assigned to you or requiring your review)
- `gg team review` - Show team-wide workload overview with active MR counts per member
- `gg mr roulette [MR_URL]` - Analyze team workload and suggest optimal assignee and reviewer for a merge request

### Webhook Server

**Install:**
```bash
go install github.com/denchenko/gg/cmd/hook@latest
```

The webhook server automatically assigns assignees and reviewers to merge requests when they are opened (excluding draft MRs and those already assigned).

**Running the server:**
```bash
export GG_TOKEN="your-gitlab-token"
export GG_TEAM="alice,bob,charlie"
./bin/hook
```

The server will start on port `8080` by default and listen for webhooks at `/gitlab/hook`.

**GitLab Webhook Configuration:**

1. Go to your GitLab project → Settings → Webhooks
2. Add a new webhook with URL: `http://your-server:8080/gitlab/hook`
3. Select "Merge request events" trigger
4. Save the webhook