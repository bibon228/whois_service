package models

import "time"

type StatusLog struct {
	ID             int       `json:"id"`
	DomainID       int       `json:"domain_id"`
	IsOnline       bool      `json:"is_online"`
	ResponseTimeMs int       `json:"response_time_ms"`
	HTTPStatus     int       `json:"http_status"`
	CheckedAt      time.Time `json:"checked_at"`
}

// DashboardStats — статистика для админ-панели
type DashboardStats struct {
	TotalDomains  int                `json:"total_domains"`
	OnlineDomains int                `json:"online_domains"`
	OfflineDomains int               `json:"offline_domains"`
	ExpiringSoon  []DomainWithStatus `json:"expiring_soon"`
}
