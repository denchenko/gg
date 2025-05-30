package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/denchenko/gg/internal/core/app"
	"github.com/denchenko/gg/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_handleGitLabWebhook(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		payload        WebhookPayload
		setupMock      func(*mocks.MockRepository) *app.App
		expectedStatus int
	}{
		{
			name:   "wrong method",
			method: http.MethodGet,
			payload: WebhookPayload{
				ObjectKind: "merge_request",
			},
			setupMock: func(m *mocks.MockRepository) *app.App {
				return &app.App{}
			},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "not merge request",
			method: http.MethodPost,
			payload: WebhookPayload{
				ObjectKind: "push",
			},
			setupMock: func(m *mocks.MockRepository) *app.App {
				return &app.App{}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "work in progress",
			method: http.MethodPost,
			payload: WebhookPayload{
				ObjectKind: "merge_request",
				ObjectAttributes: struct {
					ID             int    `json:"iid"`
					State          string `json:"state"`
					Title          string `json:"title"`
					Description    string `json:"description"`
					WorkInProgress bool   `json:"work_in_progress"`
					AssigneeID     int    `json:"assignee_id"`
					AuthorID       int    `json:"author_id"`
				}{
					WorkInProgress: true,
				},
			},
			setupMock: func(m *mocks.MockRepository) *app.App {
				return &app.App{}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "has assignee",
			method: http.MethodPost,
			payload: WebhookPayload{
				ObjectKind: "merge_request",
				Project: struct {
					ID int `json:"id"`
				}{ID: 1},
				ObjectAttributes: struct {
					ID             int    `json:"iid"`
					State          string `json:"state"`
					Title          string `json:"title"`
					Description    string `json:"description"`
					WorkInProgress bool   `json:"work_in_progress"`
					AssigneeID     int    `json:"assignee_id"`
					AuthorID       int    `json:"author_id"`
				}{
					ID:         1,
					AssigneeID: 1,
				},
			},
			setupMock: func(m *mocks.MockRepository) *app.App {
				return &app.App{}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "invalid JSON",
			method:  http.MethodPost,
			payload: WebhookPayload{},
			setupMock: func(m *mocks.MockRepository) *app.App {
				return &app.App{}
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mocks.MockRepository{}
			appInstance := tt.setupMock(repo)

			server := &Server{
				app: appInstance,
			}

			var body bytes.Buffer
			if tt.name == "invalid JSON" {
				body.WriteString("invalid json")
			} else {
				require.NoError(t, json.NewEncoder(&body).Encode(tt.payload))
			}

			req := httptest.NewRequest(tt.method, "/gitlab/hook", &body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleGitLabWebhook(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
