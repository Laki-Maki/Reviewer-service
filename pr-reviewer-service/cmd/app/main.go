package main

import (
	"net/http"
	"pr-reviewer-service/internal/db"
	"pr-reviewer-service/internal/handlers"
	"pr-reviewer-service/internal/logger"
	"pr-reviewer-service/internal/repositories"
	"pr-reviewer-service/internal/services"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	// Инициализация логгера
	logger.Init()
	defer logger.Logger.Sync() // Сбрасываем буфер на случай использования асинхронного логирования

	logger.Logger.Info("Starting PR Reviewer Service...")

	// Подключение к базе данных
	database, err := db.New()
	if err != nil {
		logger.Logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Conn.Close()
	logger.Logger.Info("Connected to PostgreSQL database successfully")

	// Репозитории
	teamRepo := repositories.NewTeamRepository(database.Conn)
	userRepo := repositories.NewUserRepository(database.Conn)
	prRepo := repositories.NewPRRepository(database.Conn)
	logger.Logger.Info("Repositories initialized")

	// Сервисы
	teamService := services.NewTeamService(teamRepo)
	userService := services.NewUserService(userRepo, prRepo)
	prService := services.NewPRService(prRepo, userRepo, teamRepo)
	logger.Logger.Info("Services initialized")

	// Handlers и маршруты
	r := chi.NewRouter()
	handlers.RegisterTeamRoutes(r, teamService)
	handlers.RegisterUserRoutes(r, userService)
	handlers.RegisterPRRoutes(r, prService)
	logger.Logger.Info("HTTP routes registered")

	// Запуск сервера
	addr := ":8080"
	logger.Logger.Info("Starting HTTP server", zap.String("address", addr))
	if err := http.ListenAndServe(addr, r); err != nil {
		logger.Logger.Fatal("Server failed", zap.Error(err))
	}
}
