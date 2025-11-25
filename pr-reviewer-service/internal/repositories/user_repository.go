package repositories

import (
	"database/sql"
	"pr-reviewer-service/internal/logger"
	"pr-reviewer-service/internal/models"
	"time"

	"go.uber.org/zap"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) SetIsActive(userID int, isActive bool) (*models.User, error) {
	_, err := r.db.Exec("UPDATE users SET is_active=$1 WHERE id=$2", isActive, userID)
	if err != nil {
		logger.Logger.Error("Failed to update user active status", zap.Error(err), zap.Int("user_id", userID), zap.Bool("is_active", isActive))
		return nil, err
	}
	logger.Logger.Info("Updated user active status", zap.Int("user_id", userID), zap.Bool("is_active", isActive))

	var user models.User
	err = r.db.QueryRow(`
		SELECT u.id, u.name, t.name, u.is_active
		FROM users u
		LEFT JOIN team_members tm ON tm.user_id=u.id
		LEFT JOIN teams t ON t.id=tm.team_id
		WHERE u.id=$1`, userID).Scan(&user.ID, &user.Username, &user.TeamName, &user.IsActive)
	if err != nil {
		logger.Logger.Error("Failed to retrieve user after update", zap.Error(err), zap.Int("user_id", userID))
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetAssignedPRs(userID int) ([]models.PullRequest, error) {
	rows, err := r.db.Query(`
        SELECT pr.id, pr.title, pr.author_id, pr.status, pr.created_at, pr.merged_at,
               prr_all.reviewer_id
        FROM pull_requests pr
        JOIN pr_reviewers prr ON pr.id = prr.pr_id
        LEFT JOIN pr_reviewers prr_all ON pr.id = prr_all.pr_id
        WHERE prr.reviewer_id = $1
        ORDER BY pr.id
    `, userID)
	if err != nil {
		logger.Logger.Error("Failed to query assigned PRs", zap.Error(err), zap.Int("user_id", userID))
		return nil, err
	}
	defer rows.Close()

	prMap := make(map[int]*models.PullRequest)
	for rows.Next() {
		var prID, authorID, reviewerID sql.NullInt64
		var title, status sql.NullString
		var createdAt, mergedAt sql.NullTime

		if err := rows.Scan(&prID, &title, &authorID, &status, &createdAt, &mergedAt, &reviewerID); err != nil {
			logger.Logger.Error("Failed to scan assigned PR", zap.Error(err), zap.Int("user_id", userID))
			return nil, err
		}

		pid := int(prID.Int64)
authorInt := int(authorID.Int64)

var mergedPtr *time.Time
if mergedAt.Valid {
    mergedPtr = &mergedAt.Time
}

if _, exists := prMap[pid]; !exists {
    prMap[pid] = &models.PullRequest{
        ID:                pid,
        Title:             title.String,
        AuthorID:          authorInt,
        Status:            status.String,
        CreatedAt:         createdAt.Time,
        MergedAt:          mergedPtr,
        AssignedReviewers: []int{},
    }
}

if reviewerID.Valid {
    prMap[pid].AssignedReviewers = append(prMap[pid].AssignedReviewers, int(reviewerID.Int64))
}


	}

	prs := make([]models.PullRequest, 0, len(prMap))
	for _, pr := range prMap {
		prs = append(prs, *pr)
	}

	logger.Logger.Info("Retrieved assigned PRs", zap.Int("user_id", userID), zap.Int("prs_count", len(prs)))
	return prs, nil
}
