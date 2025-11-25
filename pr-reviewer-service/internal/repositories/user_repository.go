package repositories

import (
	"database/sql"
	"pr-reviewer-service/internal/logger"
	"pr-reviewer-service/internal/models"

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
        SELECT pr.id, pr.title, pr.author_id, pr.status, pr.created_at, pr.merged_at
        FROM pull_requests pr
        JOIN pr_reviewers prr ON pr.id = prr.pr_id
        WHERE prr.reviewer_id = $1
    `, userID)
	if err != nil {
		logger.Logger.Error("Failed to query assigned PRs", zap.Error(err), zap.Int("user_id", userID))
		return nil, err
	}
	defer rows.Close()

	var prs []models.PullRequest
	for rows.Next() {
		var pr models.PullRequest
		if err := rows.Scan(&pr.ID, &pr.Title, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &pr.MergedAt); err != nil {
			logger.Logger.Error("Failed to scan assigned PR", zap.Error(err), zap.Int("user_id", userID))
			return nil, err
		}

		// Получаем AssignedReviewers
		revRows, err := r.db.Query("SELECT reviewer_id FROM pr_reviewers WHERE pr_id=$1", pr.ID)
		if err != nil {
			logger.Logger.Error("Failed to query reviewers for PR", zap.Error(err), zap.Int("pr_id", pr.ID))
			return nil, err
		}
		for revRows.Next() {
			var revID int
			if err := revRows.Scan(&revID); err != nil {
				revRows.Close()
				logger.Logger.Error("Failed to scan reviewer ID", zap.Error(err), zap.Int("pr_id", pr.ID))
				return nil, err
			}
			pr.AssignedReviewers = append(pr.AssignedReviewers, revID)
		}
		revRows.Close()

		prs = append(prs, pr)
	}

	logger.Logger.Info("Retrieved assigned PRs", zap.Int("user_id", userID), zap.Int("prs_count", len(prs)))
	return prs, nil
}
