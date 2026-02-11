package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email,omitempty"`
	Password  string    `json:"-"`
	IsActive  bool      `json:"is_active"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewUser(username, passwordHash string) *User {
	now := time.Now()
	return &User{
		ID:        uuid.New(),
		Username:  username,
		Password:  passwordHash,
		IsActive:  true,
		IsAdmin:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
