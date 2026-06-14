package main

import "os"

// Config хранит все настройки приложения
var AppConfig = struct {
	DSN       string
	JWTSecret string
	Port      string
}{
	DSN:       getEnv("DSN", "postgres://postgres:postgres@localhost/whois_db?sslmode=disable"),
	JWTSecret: getEnv("JWT_SECRET", "whois-super-secret-jwt-key-2024"),
	Port:      getEnv("PORT", ":8080"),
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
