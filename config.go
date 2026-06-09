package main

// Config хранит все настройки приложения
var AppConfig = struct {
	DSN       string
	JWTSecret string
	Port      string
}{
	DSN:       "postgres://postgres:postgres@localhost/whois_db?sslmode=disable",
	JWTSecret: "whois-super-secret-jwt-key-2024",
	Port:      ":8080",
}
