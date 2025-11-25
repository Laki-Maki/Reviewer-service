package services

import (
	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/repositories"
)

type UserService struct {
	userRepo *repositories.UserRepository
	prRepo   *repositories.PRRepository
}

func NewUserService(userRepo *repositories.UserRepository, prRepo *repositories.PRRepository) *UserService {
	return &UserService{userRepo: userRepo, prRepo: prRepo}
}

func (s *UserService) SetIsActive(userID int, isActive bool) (*models.User, error) {
	return s.userRepo.SetIsActive(userID, isActive)
}

func (s *UserService) GetReview(userID int) ([]models.PullRequest, error) {
	return s.userRepo.GetAssignedPRs(userID)
}
