package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"pr-reviewer-service/internal/logger"
	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/services"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func RegisterUserRoutes(r chi.Router, svc *services.UserService) {
	r.Post("/users/setIsActive", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			UserID   int  `json:"user_id"`
			IsActive bool `json:"is_active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Logger.Warn("Failed to decode SetIsActive request", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			resp := models.ErrorResponse{Error: models.ErrorDetail{Code: "BAD_REQUEST", Message: err.Error()}}
			json.NewEncoder(w).Encode(resp)
			return
		}

		user, err := svc.SetIsActive(req.UserID, req.IsActive)
		if err != nil {
			logger.Logger.Error("Failed to set user active status", zap.Error(err), zap.Int("user_id", req.UserID), zap.Bool("is_active", req.IsActive))
			w.WriteHeader(http.StatusNotFound)
			resp := models.ErrorResponse{Error: models.ErrorDetail{
				Code:    "NOT_FOUND",
				Message: fmt.Sprintf("user with id %d not found", req.UserID),
			}}
			json.NewEncoder(w).Encode(resp)
			return
		}

		logger.Logger.Info("User active status updated",
			zap.Int("user_id", user.ID),
			zap.Bool("is_active", user.IsActive),
		)
		json.NewEncoder(w).Encode(map[string]interface{}{"user": user})
	})

	r.Get("/users/getReview", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		idStr := r.URL.Query().Get("user_id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			logger.Logger.Warn("Invalid user_id in GetReview request", zap.String("user_id", idStr), zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			resp := models.ErrorResponse{Error: models.ErrorDetail{Code: "BAD_REQUEST", Message: "invalid user_id"}}
			json.NewEncoder(w).Encode(resp)
			return
		}

		prs, err := svc.GetReview(id)
		if err != nil {
			logger.Logger.Error("Failed to get assigned PRs", zap.Error(err), zap.Int("user_id", id))
			w.WriteHeader(http.StatusNotFound)
			resp := models.ErrorResponse{Error: models.ErrorDetail{
				Code:    "NOT_FOUND",
				Message: fmt.Sprintf("user with id %d not found", id),
			}}
			json.NewEncoder(w).Encode(resp)
			return
		}

		logger.Logger.Info("Retrieved assigned PRs for user", zap.Int("user_id", id), zap.Int("count", len(prs)))
		json.NewEncoder(w).Encode(map[string]interface{}{"user_id": id, "pull_requests": prs})
	})
}
