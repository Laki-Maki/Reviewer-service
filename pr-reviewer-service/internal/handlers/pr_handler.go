package handlers

import (
	"encoding/json"
	"net/http"
	"pr-reviewer-service/internal/logger"
	"pr-reviewer-service/internal/services"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func RegisterPRRoutes(r chi.Router, svc *services.PRService) {
	r.Post("/pullRequest/create", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Title    string `json:"pull_request_name"`
			AuthorID int    `json:"author_id"`
			TeamID   int    `json:"team_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Logger.Warn("Failed to decode CreatePR request", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		pr, err := svc.CreatePR(req.Title, req.AuthorID, req.TeamID)
		if err != nil {
			logger.Logger.Error("Failed to create PR", zap.Error(err), zap.Int("author_id", req.AuthorID))
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		logger.Logger.Info("Created new Pull Request", zap.Int("pr_id", pr.ID), zap.Int("author_id", req.AuthorID))
		json.NewEncoder(w).Encode(map[string]interface{}{"pr": pr})
	})

	r.Post("/pullRequest/merge", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PRID int `json:"pull_request_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Logger.Warn("Failed to decode MergePR request", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		pr, err := svc.MergePR(req.PRID)
		if err != nil {
			logger.Logger.Error("Failed to merge PR", zap.Error(err), zap.Int("pr_id", req.PRID))
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		logger.Logger.Info("Merged Pull Request", zap.Int("pr_id", pr.ID))
		json.NewEncoder(w).Encode(map[string]interface{}{"pr": pr})
	})

	r.Post("/pullRequest/reassign", func(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PRID      int `json:"pull_request_id"`
		OldUserID int `json:"old_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Logger.Warn("Failed to decode ReassignPR request", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// ReassignReviewer возвращает: PR, новый ревьювер, ошибка
	pr, newReviewerID, err := svc.ReassignReviewer(req.PRID, req.OldUserID)
	if err != nil {
		logger.Logger.Error("Failed to reassign reviewer", zap.Error(err),
			zap.Int("pr_id", req.PRID), zap.Int("old_user_id", req.OldUserID))
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	logger.Logger.Info("Reassigned PR reviewer",
		zap.Int("pr_id", pr.ID),
		zap.Int("old_user_id", req.OldUserID),
		zap.Int("new_user_id", newReviewerID),
	)
	json.NewEncoder(w).Encode(map[string]interface{}{"pr": pr, "replaced_by": newReviewerID})
})

}
