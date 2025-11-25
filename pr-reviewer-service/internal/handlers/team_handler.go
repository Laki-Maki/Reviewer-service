package handlers

import (
	"encoding/json"
	"net/http"
	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/services"
	"pr-reviewer-service/internal/logger"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

)

func RegisterTeamRoutes(r chi.Router, svc *services.TeamService) {
	// Создание команды
	r.Post("/team/add", func(w http.ResponseWriter, r *http.Request) {
		var team models.Team
		if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
			logger.Logger.Error("Failed to decode request body", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := svc.AddTeam(&team); err != nil {
			logger.Logger.Error("Failed to add team", zap.Error(err), zap.String("team_name", team.TeamName))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		logger.Logger.Info("Added new team", zap.String("team_name", team.TeamName))
		json.NewEncoder(w).Encode(map[string]interface{}{"team": team})
	})

	// Получение команды
	r.Get("/team/get", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("team_name")
		team, err := svc.GetTeam(name)
		if err != nil {
			logger.Logger.Warn("Team not found", zap.String("team_name", name), zap.Error(err))
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		logger.Logger.Info("Fetched team", zap.String("team_name", name))
		json.NewEncoder(w).Encode(team)
	})
}
