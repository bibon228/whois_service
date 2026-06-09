package models

import "time"

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	AcceptedTerms bool   `json:"accepted_terms"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
