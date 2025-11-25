package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"pr-reviewer-service/internal/logger"
	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/services"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func RegisterPRRoutes(r chi.Router, svc *services.PRService) {
	r.Post("/pullRequest/create", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			Title    string `json:"pull_request_name"`
			AuthorID int    `json:"author_id"`
			TeamID   int    `json:"team_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Logger.Warn("Failed to decode CreatePR request", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			resp := models.ErrorResponse{Error: models.ErrorDetail{Code: "BAD_REQUEST", Message: err.Error()}}
			json.NewEncoder(w).Encode(resp)
			return
		}

		pr, err := svc.CreatePR(req.Title, req.AuthorID, req.TeamID)
		if err != nil {
			logger.Logger.Error("Failed to create PR", zap.Error(err), zap.Int("author_id", req.AuthorID))
			w.WriteHeader(http.StatusConflict)
			resp := models.ErrorResponse{Error: models.ErrorDetail{Code: "PR_EXISTS", Message: err.Error()}}
			json.NewEncoder(w).Encode(resp)
			return
		}

		w.WriteHeader(http.StatusCreated)
		logger.Logger.Info("Created new Pull Request", zap.Int("pr_id", pr.ID), zap.Int("author_id", req.AuthorID))
		json.NewEncoder(w).Encode(map[string]interface{}{"pr": pr})
	})

	r.Post("/pullRequest/merge", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			PRID int `json:"pull_request_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Logger.Warn("Failed to decode MergePR request", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			resp := models.ErrorResponse{Error: models.ErrorDetail{Code: "BAD_REQUEST", Message: err.Error()}}
			json.NewEncoder(w).Encode(resp)
			return
		}

		pr, err := svc.MergePR(req.PRID)
		if err != nil {
			logger.Logger.Error("Failed to merge PR", zap.Error(err), zap.Int("pr_id", req.PRID))
			w.WriteHeader(http.StatusNotFound)
			resp := models.ErrorResponse{Error: models.ErrorDetail{
				Code:    "NOT_FOUND",
				Message: fmt.Sprintf("pull request with id %d not found", req.PRID),
			}}
			json.NewEncoder(w).Encode(resp)
			return
		}

		logger.Logger.Info("Merged Pull Request", zap.Int("pr_id", pr.ID))
		json.NewEncoder(w).Encode(map[string]interface{}{"pr": pr})
	})

	r.Post("/pullRequest/reassign", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			PRID      int `json:"pull_request_id"`
			OldUserID int `json:"old_user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Logger.Warn("Failed to decode ReassignPR request", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			resp := models.ErrorResponse{Error: models.ErrorDetail{Code: "BAD_REQUEST", Message: err.Error()}}
			json.NewEncoder(w).Encode(resp)
			return
		}

		pr, newReviewerID, err := svc.ReassignReviewer(req.PRID, req.OldUserID)
		if err != nil {
			logger.Logger.Error("Failed to reassign reviewer", zap.Error(err),
				zap.Int("pr_id", req.PRID), zap.Int("old_user_id", req.OldUserID))
			w.WriteHeader(http.StatusConflict)
			resp := models.ErrorResponse{Error: models.ErrorDetail{Code: "NO_CANDIDATE", Message: err.Error()}}
			json.NewEncoder(w).Encode(resp)
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
