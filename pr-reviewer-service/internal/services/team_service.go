package services

import (
	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/repositories"
)

type TeamService struct {
	repo *repositories.TeamRepository
}

func NewTeamService(repo *repositories.TeamRepository) *TeamService {
	return &TeamService{repo: repo}
}

func (s *TeamService) AddTeam(team *models.Team) error {
	return s.repo.CreateTeam(team)
}

func (s *TeamService) GetTeam(name string) (*models.Team, error) {
	return s.repo.GetTeam(name)
}
