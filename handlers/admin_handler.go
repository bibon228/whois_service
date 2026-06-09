package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"whois_service/models"
	"whois_service/service"
	"whois_service/worker"
)

type AdminHandler struct {
	domainService *service.DomainService
	whoisService  *service.WhoisService
	monitorWorker *worker.MonitorWorker
}

func NewAdminHandler(ds *service.DomainService, ws *service.WhoisService, mw *worker.MonitorWorker) *AdminHandler {
	return &AdminHandler{domainService: ds, whoisService: ws, monitorWorker: mw}
}

// Stats — GET /api/admin/stats — статистика для дашборда
func (h *AdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.domainService.GetStats()
	if err != nil {
		JSONError(w, "failed to fetch stats", http.StatusInternalServerError)
		return
	}
	JSONResponse(w, stats, http.StatusOK)
}

// CreateDomain — POST /api/admin/domains — добавить домен
func (h *AdminHandler) CreateDomain(w http.ResponseWriter, r *http.Request) {
	var req models.CreateDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		JSONError(w, "domain name is required", http.StatusBadRequest)
		return
	}

	domain, err := h.domainService.CreateDomain(req.Name)
	if err != nil {
		JSONError(w, "domain already exists or invalid", http.StatusConflict)
		return
	}

	// Триггерим немедленную проверку WHOIS + статуса
	h.monitorWorker.TriggerCheck(domain.ID, domain.Name)

	JSONResponse(w, domain, http.StatusCreated)
}

// UpdateDomain — PUT /api/admin/domains/{id} — обновить домен
func (h *AdminHandler) UpdateDomain(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		JSONError(w, "invalid domain id", http.StatusBadRequest)
		return
	}

	var req models.UpdateDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "invalid request", http.StatusBadRequest)
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	domain, err := h.domainService.UpdateDomain(id, req.Name, isActive)
	if err != nil {
		JSONError(w, "failed to update domain", http.StatusInternalServerError)
		return
	}

	JSONResponse(w, domain, http.StatusOK)
}

// DeleteDomain — DELETE /api/admin/domains/{id} — удалить домен
func (h *AdminHandler) DeleteDomain(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		JSONError(w, "invalid domain id", http.StatusBadRequest)
		return
	}

	if err := h.domainService.DeleteDomain(id); err != nil {
		JSONError(w, "failed to delete domain", http.StatusInternalServerError)
		return
	}

	JSONResponse(w, map[string]string{"status": "deleted"}, http.StatusOK)
}

// RefreshDomain — POST /api/admin/domains/{id}/refresh — принудительно обновить WHOIS
func (h *AdminHandler) RefreshDomain(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		JSONError(w, "invalid domain id", http.StatusBadRequest)
		return
	}

	// Запускаем обновление в горутине с контекстом, чтобы не блокировать ответ
	go func() {
		if err := h.whoisService.RefreshDomain(r.Context(), id); err != nil {
			// Ошибка логируется внутри сервиса
		}
	}()

	JSONResponse(w, map[string]string{"status": "refresh started"}, http.StatusAccepted)
}
