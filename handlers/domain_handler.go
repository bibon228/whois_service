package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"whois_service/service"
)

type DomainHandler struct {
	domainService *service.DomainService
}

func NewDomainHandler(ds *service.DomainService) *DomainHandler {
	return &DomainHandler{domainService: ds}
}

// List — GET /api/domains — список всех доменов со статусами
func (h *DomainHandler) List(w http.ResponseWriter, r *http.Request) {
	domains, err := h.domainService.ListDomains()
	if err != nil {
		JSONError(w, "failed to fetch domains", http.StatusInternalServerError)
		return
	}
	JSONResponse(w, domains, http.StatusOK)
}

// Get — GET /api/domains/{name} — один домен с полной информацией
func (h *DomainHandler) Get(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	domain, err := h.domainService.GetDomain(name)
	if err != nil {
		JSONError(w, "domain not found", http.StatusNotFound)
		return
	}
	JSONResponse(w, domain, http.StatusOK)
}

// StatusHistory — GET /api/domains/{name}/status — история статусов
func (h *DomainHandler) StatusHistory(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	history, err := h.domainService.GetStatusHistory(name, 100)
	if err != nil {
		JSONError(w, "failed to fetch status history", http.StatusInternalServerError)
		return
	}
	JSONResponse(w, history, http.StatusOK)
}

// Whois — GET /api/domains/{name}/whois — последняя WHOIS-выписка
func (h *DomainHandler) Whois(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	whois, err := h.domainService.GetWhois(name)
	if err != nil {
		JSONError(w, "no WHOIS data available", http.StatusNotFound)
		return
	}
	JSONResponse(w, whois, http.StatusOK)
}
