package http

import (
	"encoding/json"
	"log"
	"net/http"
)

// WebhookPayload represents the GitLab webhook payload.
type WebhookPayload struct {
	ObjectKind string `json:"object_kind"`
	Project    struct {
		ID int `json:"id"`
	} `json:"project"`
	ObjectAttributes struct {
		ID             int    `json:"iid"`
		State          string `json:"state"`
		Title          string `json:"title"`
		Description    string `json:"description"`
		WorkInProgress bool   `json:"work_in_progress"`
		AssigneeID     int    `json:"assignee_id"`
		AuthorID       int    `json:"author_id"`
	} `json:"object_attributes"`
}

func (s *Server) handleGitLabWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	var payload WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)

		return
	}

	if payload.ObjectKind != "merge_request" {
		w.WriteHeader(http.StatusOK)

		return
	}

	if payload.ObjectAttributes.WorkInProgress || payload.ObjectAttributes.AssigneeID != 0 {
		w.WriteHeader(http.StatusOK)

		return
	}

	mr, err := s.app.GetMergeRequest(r.Context(), payload.Project.ID, payload.ObjectAttributes.ID)
	if err != nil {
		log.Printf("Failed to get merge request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)

		return
	}

	workloads, err := s.app.AnalyzeWorkload(r.Context(), payload.Project.ID)
	if err != nil {
		log.Printf("Failed to analyze workload: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)

		return
	}

	assignee, reviewer, err := s.app.SuggestAssigneeAndReviewer(r.Context(), mr, workloads)
	if err != nil {
		log.Printf("Failed to suggest assignee and reviewer: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)

		return
	}

	if err := s.app.UpdateMergeRequest(r.Context(), mr.ProjectID, mr.IID, &assignee.ID, []int{reviewer.ID}); err != nil {
		log.Printf("Failed to update merge request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
}
