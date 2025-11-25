package handlers

import (
	"encoding/json"
	"net/http"
	"pr-reviewer-service/internal/logger"
	"pr-reviewer-service/internal/services"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func RegisterUserRoutes(r chi.Router, svc *services.UserService) {
	r.Post("/users/setIsActive", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			UserID   int  `json:"user_id"`
			IsActive bool `json:"is_active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Logger.Warn("Failed to decode SetIsActive request", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		user, err := svc.SetIsActive(req.UserID, req.IsActive)
		if err != nil {
			logger.Logger.Error("Failed to set user active status", zap.Error(err), zap.Int("user_id", req.UserID), zap.Bool("is_active", req.IsActive))
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		logger.Logger.Info("User active status updated",
			zap.Int("user_id", user.ID),
			zap.Bool("is_active", user.IsActive),
		)
		json.NewEncoder(w).Encode(map[string]interface{}{"user": user})
	})

	r.Get("/users/getReview", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("user_id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			logger.Logger.Warn("Invalid user_id in GetReview request", zap.String("user_id", idStr), zap.Error(err))
			http.Error(w, "invalid user_id", http.StatusBadRequest)
			return
		}

		prs, err := svc.GetReview(id)
		if err != nil {
			logger.Logger.Error("Failed to get assigned PRs", zap.Error(err), zap.Int("user_id", id))
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		logger.Logger.Info("Retrieved assigned PRs for user", zap.Int("user_id", id), zap.Int("count", len(prs)))
		json.NewEncoder(w).Encode(map[string]interface{}{"user_id": id, "pull_requests": prs})
	})
}
