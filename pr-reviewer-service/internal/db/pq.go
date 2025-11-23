package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
    Conn *sql.DB
}

func New() (*DB, error) {
    connStr := "postgres://user:pass@db:5432/pr_review?sslmode=disable"
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, err
    }

    // Даем больше попыток подключения
    for i := 0; i < 5; i++ {
        err = db.Ping()
        if err == nil {
            break
        }
        log.Printf("Waiting for database... attempt %d", i+1)
        time.Sleep(2 * time.Second) // раскомментировать если нужно
    }

    if err != nil {
        return nil, fmt.Errorf("failed to connect to database after retries: %w", err)
    }

    fmt.Println("✅ Connected to PostgreSQL!")
    return &DB{Conn: db}, nil
}