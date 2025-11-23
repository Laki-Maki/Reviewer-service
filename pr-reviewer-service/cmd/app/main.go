package main

import (
    "log"
    "pr-reviewer-service/internal/db"
)

func main() {
    db, err := db.New()
    if err != nil {
        log.Fatal("❌ Database connection failed:", err)
    }
    defer db.Conn.Close()

    // Простой запрос чтобы проверить что таблицы создались
    var teamCount int
    err = db.Conn.QueryRow("SELECT COUNT(*) FROM teams").Scan(&teamCount)
    if err != nil {
        log.Fatal("❌ Query failed:", err)
    }

    log.Printf("✅ Database is ready! Found %d teams", teamCount)
    log.Println("🚀 Service started successfully!")
    createTestPR(db)
}

func createTestPR(db *db.DB) {
    // Создаём PR от Alice (id = 1) в команде Backend (id = 1)
    var prID int
    err := db.Conn.QueryRow(
        "INSERT INTO pull_requests (title, author_id, team_id, status) VALUES ($1,$2,$3,'OPEN') RETURNING id",
        "Test PR", 1, 1,
    ).Scan(&prID)
    if err != nil {
        log.Fatal(err)
    }

    // Назначаем до 2 активных ревьюверов (исключаем автора)
    rows, _ := db.Conn.Query("SELECT id FROM users u JOIN team_members tm ON u.id=tm.user_id WHERE tm.team_id=$1 AND u.is_active AND u.id<>$2", 1, 1)
    reviewers := []int{}
    for rows.Next() {
        var id int
        rows.Scan(&id)
        reviewers = append(reviewers, id)
    }
    for i, r := range reviewers {
        if i >= 2 {
            break
        }
        _, err := db.Conn.Exec("INSERT INTO pr_reviewers (pr_id, reviewer_id) VALUES ($1,$2)", prID, r)
        if err != nil {
            log.Fatal(err)
        }
    }

    log.Printf("PR #%d создан с ревьюверами: %v", prID, reviewers)
}
