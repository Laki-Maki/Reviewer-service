package repositories

import (
	"database/sql"
	"errors"
	"pr-reviewer-service/internal/logger"
	"pr-reviewer-service/internal/models"
	"time"

	"go.uber.org/zap"
)

type PRRepository struct {
	db *sql.DB
}

func NewPRRepository(db *sql.DB) *PRRepository {
	return &PRRepository{db: db}
}

func (r *PRRepository) CreatePR(title string, authorID int, teamID int) (int, error) {
	var prID int
	err := r.db.QueryRow(`
		INSERT INTO pull_requests(title, author_id, team_id, status)
		VALUES($1,$2,$3,'OPEN') RETURNING id`, title, authorID, teamID).Scan(&prID)
	if err != nil {
		logger.Logger.Error("Failed to create PR", zap.Error(err), zap.String("title", title), zap.Int("author_id", authorID))
		return 0, err
	}

	logger.Logger.Info("Created Pull Request", zap.Int("pr_id", prID), zap.String("title", title), zap.Int("author_id", authorID))
	return prID, nil
}

func (r *PRRepository) AssignReviewers(prID int, reviewers []int) error {
	for _, reviewerID := range reviewers {
		_, err := r.db.Exec("INSERT INTO pr_reviewers(pr_id, reviewer_id) VALUES($1,$2)", prID, reviewerID)
		if err != nil {
			logger.Logger.Error("Failed to assign reviewer", zap.Error(err), zap.Int("pr_id", prID), zap.Int("reviewer_id", reviewerID))
			return err
		}
		logger.Logger.Info("Assigned reviewer to PR", zap.Int("pr_id", prID), zap.Int("reviewer_id", reviewerID))
	}
	return nil
}

func (r *PRRepository) GetPR(prID int) (*models.PullRequest, error) {
	var pr models.PullRequest
	err := r.db.QueryRow(`
		SELECT id, title, author_id, status, created_at, merged_at 
		FROM pull_requests WHERE id=$1
	`, prID).Scan(&pr.ID, &pr.Title, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &pr.MergedAt)
	if err != nil {
		logger.Logger.Error("Failed to get PR", zap.Error(err), zap.Int("pr_id", prID))
		return nil, err
	}

	rows, err := r.db.Query("SELECT reviewer_id FROM pr_reviewers WHERE pr_id=$1", prID)
	if err != nil {
		logger.Logger.Error("Failed to get PR reviewers", zap.Error(err), zap.Int("pr_id", prID))
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			logger.Logger.Error("Failed to scan reviewer ID", zap.Error(err), zap.Int("pr_id", prID))
			return nil, err
		}
		pr.AssignedReviewers = append(pr.AssignedReviewers, id)
	}

	return &pr, nil
}

func (r *PRRepository) MergePR(prID int) (*models.PullRequest, error) {
	var status string
	err := r.db.QueryRow("SELECT status FROM pull_requests WHERE id=$1", prID).Scan(&status)
	if err != nil {
		logger.Logger.Error("Failed to get PR status", zap.Error(err), zap.Int("pr_id", prID))
		return nil, err
	}
	if status == "MERGED" {
		return r.GetPR(prID)
	}

	now := time.Now()
	_, err = r.db.Exec("UPDATE pull_requests SET status='MERGED', merged_at=$1 WHERE id=$2", now, prID)
	if err != nil {
		logger.Logger.Error("Failed to merge PR", zap.Error(err), zap.Int("pr_id", prID))
		return nil, err
	}

	logger.Logger.Info("Merged Pull Request", zap.Int("pr_id", prID))
	return r.GetPR(prID)
}

func (r *PRRepository) ReassignReviewer(prID int, oldReviewerID int) (int, error) {
	var status string
	err := r.db.QueryRow("SELECT status FROM pull_requests WHERE id=$1", prID).Scan(&status)
	if err != nil {
		logger.Logger.Error("Failed to get PR status for reassignment", zap.Error(err), zap.Int("pr_id", prID))
		return 0, err
	}
	if status == "MERGED" {
		return 0, errors.New("cannot reassign reviewers on merged PR")
	}

	var teamID int
	err = r.db.QueryRow("SELECT team_id FROM team_members WHERE user_id=$1 LIMIT 1", oldReviewerID).Scan(&teamID)
	if err != nil {
		logger.Logger.Error("Failed to get team ID for old reviewer", zap.Error(err), zap.Int("old_reviewer_id", oldReviewerID))
		return 0, err
	}

	var newReviewerID int
	err = r.db.QueryRow(`
		SELECT u.id
		FROM users u
		JOIN team_members tm ON tm.user_id = u.id
		WHERE tm.team_id = $1 AND u.is_active=true AND u.id<>$2
		ORDER BY RANDOM() LIMIT 1
	`, teamID, oldReviewerID).Scan(&newReviewerID)
	if err != nil {
		logger.Logger.Error("Failed to select new reviewer", zap.Error(err), zap.Int("pr_id", prID))
		return 0, err
	}

	_, err = r.db.Exec("UPDATE pr_reviewers SET reviewer_id=$1 WHERE pr_id=$2 AND reviewer_id=$3",
		newReviewerID, prID, oldReviewerID)
	if err != nil {
		logger.Logger.Error("Failed to update PR reviewer", zap.Error(err), zap.Int("pr_id", prID), zap.Int("old_reviewer_id", oldReviewerID), zap.Int("new_reviewer_id", newReviewerID))
		return 0, err
	}

	logger.Logger.Info("Reassigned PR reviewer", zap.Int("pr_id", prID), zap.Int("old_reviewer_id", oldReviewerID), zap.Int("new_reviewer_id", newReviewerID))
	return newReviewerID, nil
}

func (r *PRRepository) GetActiveTeamMembers(teamID int, excludeUserID int) ([]int, error) {
	rows, err := r.db.Query(`
		SELECT u.id
		FROM users u
		JOIN team_members tm ON tm.user_id = u.id
		WHERE tm.team_id = $1 AND u.id <> $2 AND u.is_active = true
	`, teamID, excludeUserID)
	if err != nil {
		logger.Logger.Error("Failed to get active team members", zap.Error(err), zap.Int("team_id", teamID))
		return nil, err
	}
	defer rows.Close()

	var members []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			logger.Logger.Error("Failed to scan team member ID", zap.Error(err), zap.Int("team_id", teamID))
			return nil, err
		}
		members = append(members, id)
	}

	logger.Logger.Info("Retrieved active team members", zap.Int("team_id", teamID), zap.Int("exclude_user_id", excludeUserID), zap.Int("count", len(members)))
	return members, nil
}
