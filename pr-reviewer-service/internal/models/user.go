package models

import (
	"encoding/json"
	"fmt"
)

type User struct {
	ID       int    `json:"-"`
	UserID   string `json:"user_id"` // для API
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

// MarshalJSON кастомный, чтобы int ID конвертировать в string
func (u User) MarshalJSON() ([]byte, error) {
	type Alias User
	return json.Marshal(&struct {
		UserID string `json:"user_id"`
		Alias
	}{
		UserID: fmt.Sprintf("u%d", u.ID),
		Alias:  (Alias)(u),
	})
}
