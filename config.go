package main

import (
	"os"
	"strings"
)

// Config хранит все настройки приложения
var AppConfig = struct {
	DSN       string
	JWTSecret string
	Port      string
}{
	DSN:       getDSN(),
	JWTSecret: getEnv("JWT_SECRET", "whois-super-secret-jwt-key-2024"),
	Port:      getPort(),
}

// getDSN возвращает строку подключения к БД.
// Railway предоставляет DATABASE_URL, локально используется DSN.
func getDSN() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	return getEnv("DSN", "postgres://postgres:postgres@localhost/whois_db?sslmode=disable")
}

// getPort возвращает порт в формате ":8080".
// Railway задаёт PORT="8080" (без двоеточия), локально может быть ":8080".
func getPort() string {
	port := getEnv("PORT", "8080")
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	return port
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
