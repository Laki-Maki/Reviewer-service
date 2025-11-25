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

func RegisterTeamRoutes(r chi.Router, svc *services.TeamService) {
	r.Post("/team/add", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var team models.Team
		if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
			logger.Logger.Error("Failed to decode request body", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			resp := models.ErrorResponse{Error: models.ErrorDetail{Code: "BAD_REQUEST", Message: err.Error()}}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if err := svc.AddTeam(&team); err != nil {
			logger.Logger.Error("Failed to add team", zap.Error(err), zap.String("team_name", team.TeamName))
			w.WriteHeader(http.StatusBadRequest)
			resp := models.ErrorResponse{Error: models.ErrorDetail{Code: "TEAM_EXISTS", Message: err.Error()}}
			json.NewEncoder(w).Encode(resp)
			return
		}

		w.WriteHeader(http.StatusCreated)
		logger.Logger.Info("Added new team", zap.String("team_name", team.TeamName))
		json.NewEncoder(w).Encode(map[string]interface{}{"team": team})
	})

	r.Get("/team/get", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		name := r.URL.Query().Get("team_name")
		team, err := svc.GetTeam(name)
		if err != nil {
			logger.Logger.Warn("Team not found", zap.String("team_name", name), zap.Error(err))
			w.WriteHeader(http.StatusNotFound)
			resp := models.ErrorResponse{Error: models.ErrorDetail{
				Code:    "NOT_FOUND",
				Message: fmt.Sprintf("team '%s' not found", name),
			}}
			json.NewEncoder(w).Encode(resp)
			return
		}

		logger.Logger.Info("Fetched team", zap.String("team_name", name))
		json.NewEncoder(w).Encode(map[string]interface{}{"team": team})
	})
}
