package models

import (
	"encoding/json"
	"fmt"
	"time"
)

type PullRequest struct {
	ID                int        `json:"id"`
	Title             string     `json:"title"`
	AuthorID          int        `json:"author_id"`
	Status            string     `json:"status"` // OPEN|MERGED
	AssignedReviewers []int      `json:"assigned_reviewers"`
	CreatedAt         time.Time  `json:"created_at,omitempty"`
	MergedAt          *time.Time `json:"merged_at,omitempty"`
}

// Кастомный MarshalJSON: преобразует ID-шники в формат API
func (pr PullRequest) MarshalJSON() ([]byte, error) {
	type Alias PullRequest

	// Преобразуем []int → []string, формат "u<ID>"
	reviewers := make([]string, len(pr.AssignedReviewers))
	for i, r := range pr.AssignedReviewers {
		reviewers[i] = fmt.Sprintf("u%d", r)
	}

	// MergedAt → string
	var mergedAt string
	if pr.MergedAt != nil {
		mergedAt = pr.MergedAt.UTC().Format(time.RFC3339)
	}

	return json.Marshal(&struct {
		PullRequestID     string   `json:"pull_request_id"`
		AuthorID          string   `json:"author_id"`
		AssignedReviewers []string `json:"assigned_reviewers"`
		CreatedAt         string   `json:"createdAt,omitempty"`
		MergedAt          string   `json:"mergedAt,omitempty"`
		Alias
	}{
		PullRequestID:     fmt.Sprintf("pr-%d", pr.ID),
		AuthorID:          fmt.Sprintf("u%d", pr.AuthorID),
		AssignedReviewers: reviewers,
		CreatedAt:         pr.CreatedAt.UTC().Format(time.RFC3339),
		MergedAt:          mergedAt,
		Alias:             (Alias)(pr),
	})
}
