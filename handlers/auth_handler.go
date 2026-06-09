package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"time"

	"whois_service/models"
	"whois_service/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(as *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: as}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		JSONError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Устанавливаем JWT в HttpOnly cookie (защита от XSS)
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    resp.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Поставь true в production (HTTPS)
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(72 * time.Hour),
	})

	JSONResponse(w, map[string]interface{}{
		"message": "success",
		"user":    resp.User,
	}, http.StatusOK)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Проверка принятия политики конфиденциальности
	if !req.AcceptedTerms {
		JSONError(w, "необходимо принять Политику конфиденциальности", http.StatusBadRequest)
		return
	}

	if len(req.Username) < 3 {
		JSONError(w, "имя пользователя минимум 3 символа", http.StatusBadRequest)
		return
	}

	// Строгая валидация пароля: мин. 8 символов, хотя бы 1 буква и 1 цифра
	if len(req.Password) < 8 {
		JSONError(w, "пароль должен содержать минимум 8 символов", http.StatusBadRequest)
		return
	}
	hasLetter := regexp.MustCompile(`[a-zA-Zа-яА-Я]`).MatchString(req.Password)
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(req.Password)
	if !hasLetter || !hasDigit {
		JSONError(w, "пароль должен содержать хотя бы одну букву и одну цифру", http.StatusBadRequest)
		return
	}

	resp, err := h.authService.Register(req.Username, req.Password)
	if err != nil {
		JSONError(w, err.Error(), http.StatusConflict)
		return
	}

	// Устанавливаем JWT в HttpOnly cookie (защита от XSS)
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    resp.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Поставь true в production (HTTPS)
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(72 * time.Hour),
	})

	JSONResponse(w, map[string]interface{}{
		"message": "success",
		"user":    resp.User,
	}, http.StatusCreated)
}

// === Помощники для JSON-ответов (используются всеми хендлерами) ===

func JSONResponse(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func JSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
