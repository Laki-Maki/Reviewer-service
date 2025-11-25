package services

import (
	"math/rand"
	"pr-reviewer-service/internal/logger"
	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/repositories"
	"time"

	"go.uber.org/zap"
)

type PRService struct {
	prRepo   *repositories.PRRepository
	userRepo *repositories.UserRepository
	teamRepo *repositories.TeamRepository
}

func NewPRService(prRepo *repositories.PRRepository, userRepo *repositories.UserRepository, teamRepo *repositories.TeamRepository) *PRService {
	return &PRService{prRepo: prRepo, userRepo: userRepo, teamRepo: teamRepo}
}

func (s *PRService) CreatePR(title string, authorID int, teamID int) (*models.PullRequest, error) {
	logger.Logger.Info("Creating Pull Request", zap.String("title", title), zap.Int("author_id", authorID), zap.Int("team_id", teamID))

	prID, err := s.prRepo.CreatePR(title, authorID, teamID)
	if err != nil {
		logger.Logger.Error("Failed to create PR", zap.Error(err), zap.String("title", title), zap.Int("author_id", authorID))
		return nil, err
	}

	candidates, err := s.prRepo.GetActiveTeamMembers(teamID, authorID)
	if err != nil {
		logger.Logger.Error("Failed to get active team members", zap.Error(err), zap.Int("team_id", teamID))
		return nil, err
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })

	if len(candidates) > 2 {
		candidates = candidates[:2]
	}

	if err := s.prRepo.AssignReviewers(prID, candidates); err != nil {
		logger.Logger.Error("Failed to assign reviewers", zap.Error(err), zap.Int("pr_id", prID))
		return nil, err
	}

	logger.Logger.Info("Successfully created PR with reviewers", zap.Int("pr_id", prID), zap.Ints("reviewer_ids", candidates))
	return s.prRepo.GetPR(prID)
}

func (s *PRService) MergePR(prID int) (*models.PullRequest, error) {
	logger.Logger.Info("Merging Pull Request", zap.Int("pr_id", prID))

	pr, err := s.prRepo.MergePR(prID)
	if err != nil {
		logger.Logger.Error("Failed to merge PR", zap.Error(err), zap.Int("pr_id", prID))
		return nil, err
	}

	logger.Logger.Info("Successfully merged PR", zap.Int("pr_id", prID))
	return pr, nil
}

func (s *PRService) ReassignReviewer(prID int, oldReviewerID int) (*models.PullRequest, int, error) {
	logger.Logger.Info("Reassigning reviewer", zap.Int("pr_id", prID), zap.Int("old_reviewer_id", oldReviewerID))

	newReviewerID, err := s.prRepo.ReassignReviewer(prID, oldReviewerID)
	if err != nil {
		logger.Logger.Error("Failed to reassign reviewer", zap.Error(err), zap.Int("pr_id", prID), zap.Int("old_reviewer_id", oldReviewerID))
		return nil, 0, err
	}

	pr, err := s.prRepo.GetPR(prID)
	if err != nil {
		logger.Logger.Error("Failed to fetch PR after reassigning reviewer", zap.Error(err), zap.Int("pr_id", prID))
		return nil, 0, err
	}

	logger.Logger.Info("Successfully reassigned reviewer", zap.Int("pr_id", prID), zap.Int("new_reviewer_id", newReviewerID))
	return pr, newReviewerID, nil
}
