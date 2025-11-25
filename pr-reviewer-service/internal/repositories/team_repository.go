package repositories

import (
	"database/sql"
	"pr-reviewer-service/internal/logger"
	"pr-reviewer-service/internal/models"

	"go.uber.org/zap"
)

type TeamRepository struct {
	db *sql.DB
}

func NewTeamRepository(db *sql.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) CreateTeam(team *models.Team) error {
	tx, err := r.db.Begin()
	if err != nil {
		logger.Logger.Error("Failed to begin transaction", zap.Error(err))
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 1. Создаём команду
	_, err = tx.Exec("INSERT INTO teams(name) VALUES($1) ON CONFLICT(name) DO NOTHING", team.TeamName)
	if err != nil {
		logger.Logger.Error("Failed to create team", zap.Error(err), zap.String("team_name", team.TeamName))
		return err
	}

	// Получаем team_id
	var teamID int
	err = tx.QueryRow("SELECT id FROM teams WHERE name=$1", team.TeamName).Scan(&teamID)
	if err != nil {
		logger.Logger.Error("Failed to get team_id", zap.Error(err), zap.String("team_name", team.TeamName))
		return err
	}

	// 2. Создаём/обновляем пользователей и привязываем к команде
	for _, member := range team.Members {
		_, err = tx.Exec(`
			INSERT INTO users(id,name,is_active) VALUES($1,$2,$3)
			ON CONFLICT(id) DO UPDATE SET name=$2, is_active=$3`,
			member.UserID, member.Username, member.IsActive,
		)
		if err != nil {
			logger.Logger.Error("Failed to upsert user", zap.Error(err), zap.Int("user_id", member.UserID))
			return err
		}

		_, err = tx.Exec(`
			INSERT INTO team_members(team_id,user_id)
			VALUES($1,$2) ON CONFLICT DO NOTHING`,
			teamID, member.UserID,
		)
		if err != nil {
			logger.Logger.Error("Failed to assign user to team", zap.Error(err), zap.Int("user_id", member.UserID), zap.String("team_name", team.TeamName))
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		logger.Logger.Error("Failed to commit CreateTeam transaction", zap.Error(err))
		return err
	}

	logger.Logger.Info("Created team with members", zap.String("team_name", team.TeamName), zap.Int("members_count", len(team.Members)))
	return nil
}

func (r *TeamRepository) GetTeam(name string) (*models.Team, error) {
	team := &models.Team{TeamName: name}
	rows, err := r.db.Query(`
		SELECT u.id, u.name, u.is_active
		FROM users u
		JOIN team_members tm ON tm.user_id=u.id
		JOIN teams t ON t.id=tm.team_id
		WHERE t.name=$1`, name)
	if err != nil {
		logger.Logger.Error("Failed to query team members", zap.Error(err), zap.String("team_name", name))
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m models.TeamMember
		if err := rows.Scan(&m.UserID, &m.Username, &m.IsActive); err != nil {
			logger.Logger.Error("Failed to scan team member", zap.Error(err), zap.String("team_name", name))
			return nil, err
		}
		team.Members = append(team.Members, m)
	}

	logger.Logger.Info("Retrieved team", zap.String("team_name", name), zap.Int("members_count", len(team.Members)))
	return team, nil
}
