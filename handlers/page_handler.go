package handlers

import (
	"net/http"
	"path/filepath"
)

type PageHandler struct {
	staticDir string
}

func NewPageHandler(staticDir string) *PageHandler {
	return &PageHandler{staticDir: staticDir}
}

func (h *PageHandler) Index(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(h.staticDir, "index.html"))
}

func (h *PageHandler) DomainPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(h.staticDir, "domain.html"))
}

func (h *PageHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(h.staticDir, "login.html"))
}

func (h *PageHandler) AdminPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(h.staticDir, "admin.html"))
}

func (h *PageHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(h.staticDir, "register.html"))
}
