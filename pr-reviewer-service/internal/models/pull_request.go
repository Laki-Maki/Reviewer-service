package models

import "time"

type PullRequest struct {
	ID                int       `json:"pull_request_id"`
	Title             string    `json:"pull_request_name"`
	AuthorID          int       `json:"author_id"`
	Status            string    `json:"status"` // OPEN|MERGED
	AssignedReviewers []int     `json:"assigned_reviewers"`
	CreatedAt         time.Time `json:"createdAt,omitempty"`
	MergedAt          *time.Time `json:"mergedAt,omitempty"`
}
