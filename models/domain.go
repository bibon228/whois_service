package models

import "time"

type Domain struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DomainWithStatus — домен с последним статусом и WHOIS-данными
type DomainWithStatus struct {
	Domain
	IsOnline       *bool      `json:"is_online"`
	ResponseTimeMs *int       `json:"response_time_ms,omitempty"`
	IPAddress      string     `json:"ip_address,omitempty"`
	Registrar      string     `json:"registrar,omitempty"`
	ExpiryDate     *time.Time `json:"expiry_date,omitempty"`
}

type CreateDomainRequest struct {
	Name string `json:"name"`
}

type UpdateDomainRequest struct {
	Name     string `json:"name"`
	IsActive *bool  `json:"is_active"`
}
