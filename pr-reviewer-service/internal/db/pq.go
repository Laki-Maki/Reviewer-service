package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"pr-reviewer-service/internal/logger"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type DB struct {
	Conn *sql.DB
}

func New() (*DB, error) {
	// Формируем строку подключения из переменных окружения
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		logger.Logger.Error("Failed to open database connection", zap.Error(err))
		return nil, err
	}

	// Пять попыток подключиться к базе
	for i := 1; i <= 5; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		logger.Logger.Warn("Waiting for database...", zap.Int("attempt", i), zap.Error(err))
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		logger.Logger.Fatal("Failed to connect to database after retries", zap.Error(err))
		return nil, fmt.Errorf("failed to connect to database after retries: %w", err)
	}

	logger.Logger.Info("✅ Connected to PostgreSQL database successfully")
	return &DB{Conn: db}, nil
}
