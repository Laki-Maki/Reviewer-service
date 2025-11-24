package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// Получаем нагрузку для каждого кандидата
type candidate struct {
	id   int
	load int
}

// CreatePR: атомарно создаёт PR и назначает до 2 ревьюверов с балансировкой нагрузки
func (r *PRRepository) CreatePR(title string, authorID int, teamID int) (int, error) {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		logger.Logger.Error("failed to begin tx CreatePR", zap.Error(err))
		return 0, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	// 1. Вставляем PR
	var prID int
	err = tx.QueryRowContext(ctx, `
		INSERT INTO pull_requests(title, author_id, team_id, status)
		VALUES($1,$2,$3,'OPEN') RETURNING id
	`, title, authorID, teamID).Scan(&prID)
	if err != nil {
		_ = tx.Rollback()
		logger.Logger.Error("Failed to create PR", zap.Error(err))
		return 0, err
	}

	// 2. Выбираем до 2 кандидатов с минимальной нагрузкой напрямую в SQL
	rows, err := tx.QueryContext(ctx, `
		WITH candidates AS (
			SELECT u.id
			FROM users u
			JOIN team_members tm ON tm.user_id = u.id
			WHERE tm.team_id = $1 AND u.is_active = true AND u.id <> $2
		),
		loads AS (
			SELECT reviewer_id, COUNT(*) as cnt
			FROM pr_reviewers
			GROUP BY reviewer_id
		)
		SELECT c.id
		FROM candidates c
		LEFT JOIN loads l ON c.id = l.reviewer_id
		ORDER BY COALESCE(l.cnt,0) ASC, RANDOM()
		LIMIT 2
	`, teamID, authorID)
	if err != nil {
		_ = tx.Rollback()
		logger.Logger.Error("Failed to query candidates with load", zap.Error(err))
		return 0, err
	}
	defer rows.Close()

	var selected []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			_ = tx.Rollback()
			logger.Logger.Error("Failed to scan selected reviewer", zap.Error(err))
			return 0, err
		}
		selected = append(selected, id)
	}

	// 3. Вставляем выбранных ревьюверов
	for _, reviewerID := range selected {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO pr_reviewers(pr_id, reviewer_id) VALUES($1,$2)",
			prID, reviewerID)
		if err != nil {
			_ = tx.Rollback()
			logger.Logger.Error("Failed to assign reviewer", zap.Error(err), zap.Int("pr_id", prID), zap.Int("reviewer_id", reviewerID))
			return 0, err
		}
	}

	// 4. Коммит транзакции
	if err := tx.Commit(); err != nil {
		logger.Logger.Error("Failed to commit tx CreatePR", zap.Error(err))
		return 0, err
	}

	logger.Logger.Info("Created PR with reviewers",
		zap.Int("pr_id", prID),
		zap.Ints("reviewer_ids", selected),
	)

	return prID, nil
}

// GetPR - получает PR и список ревьюверов
func (r *PRRepository) GetPR(prID int) (*models.PullRequest, error) {
	var pr models.PullRequest
	err := r.db.QueryRow(`
		SELECT id, title, author_id, status, created_at, merged_at
		FROM pull_requests WHERE id=$1
	`, prID).Scan(&pr.ID, &pr.Title, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &pr.MergedAt)
	if err != nil {
		logger.Logger.Error("Failed to get PR", zap.Error(err), zap.Int("pr_id", prID))
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("not found")
		}
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

// MergePR - идемпотентный merge
func (r *PRRepository) MergePR(prID int) (*models.PullRequest, error) {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		logger.Logger.Error("Failed to begin tx MergePR", zap.Error(err))
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	var status string
	var mergedAt sql.NullTime
	err = tx.QueryRowContext(ctx, "SELECT status, merged_at FROM pull_requests WHERE id=$1 FOR UPDATE", prID).Scan(&status, &mergedAt)
	if err != nil {
		_ = tx.Rollback()
		logger.Logger.Error("Failed to select PR for merge", zap.Error(err), zap.Int("pr_id", prID))
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("not found")
		}
		return nil, err
	}

	if status == "MERGED" {
		_ = tx.Commit()
		return r.GetPR(prID)
	}

	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, "UPDATE pull_requests SET status='MERGED', merged_at = $1 WHERE id=$2", now, prID)
	if err != nil {
		_ = tx.Rollback()
		logger.Logger.Error("Failed to update PR to MERGED", zap.Error(err), zap.Int("pr_id", prID))
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		logger.Logger.Error("Failed to commit MergePR", zap.Error(err))
		return nil, err
	}

	return r.GetPR(prID)
}

// ReassignReviewer - атомарно заменяет oldReviewerID на нового кандидата из той же команды
// Возвращает новый reviewer id
func (r *PRRepository) ReassignReviewer(prID int, oldReviewerID int) (int, error) {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		logger.Logger.Error("Failed to begin tx ReassignReviewer", zap.Error(err), zap.Int("pr_id", prID), zap.Int("old_reviewer_id", oldReviewerID))
		return 0, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	logger.Logger.Info("Starting reassignment", zap.Int("pr_id", prID), zap.Int("old_reviewer_id", oldReviewerID))

	// 1) Проверяем статус PR
	var status string
	err = tx.QueryRowContext(ctx, "SELECT status FROM pull_requests WHERE id=$1 FOR UPDATE", prID).Scan(&status)
	if err != nil {
		_ = tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			logger.Logger.Warn("PR not found", zap.Int("pr_id", prID))
			return 0, fmt.Errorf("not found")
		}
		logger.Logger.Error("Failed to get PR status", zap.Error(err), zap.Int("pr_id", prID))
		return 0, err
	}
	logger.Logger.Info("PR status retrieved", zap.Int("pr_id", prID), zap.String("status", status))
	if status == "MERGED" {
		_ = tx.Rollback()
		logger.Logger.Warn("Cannot reassign reviewer: PR already merged", zap.Int("pr_id", prID))
		return 0, fmt.Errorf("PR_MERGED: cannot reassign on merged PR")
	}

	// 2) Проверяем, что oldReviewer назначен
	var exists int
	err = tx.QueryRowContext(ctx, "SELECT 1 FROM pr_reviewers WHERE pr_id=$1 AND reviewer_id=$2", prID, oldReviewerID).Scan(&exists)
	if err != nil {
		_ = tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			logger.Logger.Warn("Old reviewer not assigned to PR", zap.Int("pr_id", prID), zap.Int("old_reviewer_id", oldReviewerID))
			return 0, fmt.Errorf("NOT_ASSIGNED: reviewer is not assigned to this PR")
		}
		logger.Logger.Error("Failed to check existing reviewer", zap.Error(err), zap.Int("pr_id", prID), zap.Int("old_reviewer_id", oldReviewerID))
		return 0, err
	}
	logger.Logger.Info("Old reviewer confirmed assigned", zap.Int("pr_id", prID), zap.Int("old_reviewer_id", oldReviewerID))

	/// 3) Получаем teamID PR, а не пользователя
	var teamID int
	err = tx.QueryRowContext(ctx, "SELECT team_id FROM pull_requests WHERE id=$1", prID).Scan(&teamID)
	if err != nil {
		_ = tx.Rollback()
		logger.Logger.Error("Failed to get team ID for PR", zap.Error(err), zap.Int("pr_id", prID))
		return 0, fmt.Errorf("not found")
	}

	// 4) Получаем список текущих ревьюеров PR для исключения
	curRows, err := tx.QueryContext(ctx, "SELECT reviewer_id FROM pr_reviewers WHERE pr_id=$1", prID)
	if err != nil {
		_ = tx.Rollback()
		logger.Logger.Error("Failed to query current reviewers", zap.Error(err), zap.Int("pr_id", prID))
		return 0, err
	}
	defer curRows.Close()
	excludeMap := map[int]struct{}{oldReviewerID: {}}
	for curRows.Next() {
		var id int
		if err := curRows.Scan(&id); err != nil {
			_ = tx.Rollback()
			logger.Logger.Error("Failed to scan current reviewer", zap.Error(err))
			return 0, err
		}
		excludeMap[id] = struct{}{}
	}
	logger.Logger.Info("Current reviewers retrieved", zap.Int("pr_id", prID), zap.Int("exclude_count", len(excludeMap)))

	// 5) Выбор кандидатов из той же команды
	rows, err := tx.QueryContext(ctx, `
		SELECT u.id
		FROM users u
		JOIN team_members tm ON tm.user_id = u.id
		WHERE tm.team_id = $1 AND u.is_active = true AND u.id <> $2
	`, teamID, oldReviewerID)
	if err != nil {
		_ = tx.Rollback()
		logger.Logger.Error("Failed to query team candidates", zap.Error(err), zap.Int("team_id", teamID))
		return 0, err
	}
	defer rows.Close()

	var candidates []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			_ = tx.Rollback()
			logger.Logger.Error("Failed to scan candidate", zap.Error(err))
			return 0, err
		}
		if _, excluded := excludeMap[id]; excluded {
			continue
		}
		candidates = append(candidates, id)
	}
	if len(candidates) == 0 {
		_ = tx.Rollback()
		logger.Logger.Warn("No active replacement candidates in team", zap.Int("team_id", teamID))
		return 0, fmt.Errorf("NO_CANDIDATE: no active replacement candidate in team")
	}
	logger.Logger.Info("Candidates retrieved", zap.Int("team_id", teamID), zap.Int("candidate_count", len(candidates)))

	// 6) Выбираем кандидата с минимальной нагрузкой
	vals := ""
	args := []interface{}{}
	for i, cid := range candidates {
		if i > 0 {
			vals += ","
		}
		args = append(args, cid)
		// Приводим к integer прямо в SQL
		vals += fmt.Sprintf("($%d::int)", i+1)
	}
	query := fmt.Sprintf(`
	WITH cands(id) AS (VALUES %s)
	SELECT cands.id
	FROM cands
	LEFT JOIN (
		SELECT reviewer_id, COUNT(*) as cnt
		FROM pr_reviewers
		GROUP BY reviewer_id
	) r ON cands.id = r.reviewer_id
	ORDER BY COALESCE(r.cnt,0) ASC, random()
	LIMIT 1
`, vals)

	var newReviewerID int
	err = tx.QueryRowContext(ctx, query, args...).Scan(&newReviewerID)
	if err != nil {
		_ = tx.Rollback()
		logger.Logger.Error("Failed to select new reviewer", zap.Error(err))
		return 0, err
	}
	logger.Logger.Info("New reviewer selected", zap.Int("pr_id", prID), zap.Int("new_reviewer_id", newReviewerID))

	// 7) Заменяем old -> new
	_, err = tx.ExecContext(ctx, "DELETE FROM pr_reviewers WHERE pr_id=$1 AND reviewer_id=$2", prID, oldReviewerID)
	if err != nil {
		_ = tx.Rollback()
		logger.Logger.Error("Failed to delete old reviewer", zap.Error(err), zap.Int("pr_id", prID), zap.Int("old_reviewer_id", oldReviewerID))
		return 0, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO pr_reviewers(pr_id, reviewer_id) VALUES($1,$2)", prID, newReviewerID)
	if err != nil {
		_ = tx.Rollback()
		logger.Logger.Error("Failed to insert new reviewer", zap.Error(err), zap.Int("pr_id", prID), zap.Int("new_reviewer_id", newReviewerID))
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		logger.Logger.Error("Failed to commit ReassignReviewer", zap.Error(err))
		return 0, err
	}

	logger.Logger.Info("Successfully reassigned PR reviewer",
		zap.Int("pr_id", prID),
		zap.Int("old_reviewer_id", oldReviewerID),
		zap.Int("new_reviewer_id", newReviewerID),
	)

	return newReviewerID, nil
}

// GetActiveTeamMembers - возвращает активных членов команды (excludeUserID может быть 0)
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
