package main

import (
	"context"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"pr-reviewer-service/internal/db"
	"pr-reviewer-service/internal/handlers"
	"pr-reviewer-service/internal/logger"
	"pr-reviewer-service/internal/repositories"
	"pr-reviewer-service/internal/services"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"pr-reviewer-service/config"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	// Инициализация логгера
	logger.Init()
	defer logger.Logger.Sync() // Сбрасываем буфер на случай использования асинхронного логирования

	logger.Logger.Info("Starting PR Reviewer Service...")

	// Подключение к базе данных
	cfg := config.Load()
	database, err := db.New(cfg)
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

	// HTTP сервер с graceful shutdown
	addr := ":" + cfg.Server.Port
	server := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Канал для ошибок сервера
	serverErrors := make(chan error, 1)

	// Запуск сервера в горутине
	go func() {
		logger.Logger.Info("Starting HTTP server", zap.String("address", addr))
		serverErrors <- server.ListenAndServe()
	}()

	// Канал для сигналов ОС
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Ждём либо сигнала, либо ошибки сервера
	select {
	case sig := <-sigChan:
		logger.Logger.Info("Received signal", zap.String("signal", sig.String()))
	case err := <-serverErrors:
		if err != http.ErrServerClosed {
			logger.Logger.Fatal("Server error", zap.Error(err))
		}
	}

	// Graceful shutdown с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Logger.Info("Shutting down server gracefully...")
	if err := server.Shutdown(ctx); err != nil {
		logger.Logger.Error("Server shutdown error", zap.Error(err))
	}

	logger.Logger.Info("Server stopped successfully")
}
