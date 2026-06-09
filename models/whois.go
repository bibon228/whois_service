package models

import "time"

type WhoisRecord struct {
	ID          int        `json:"id"`
	DomainID    int        `json:"domain_id"`
	Registrar   string     `json:"registrar"`
	Registrant  string     `json:"registrant"`
	CreatedDate *time.Time `json:"created_date"`
	ExpiryDate  *time.Time `json:"expiry_date"`
	NameServers string     `json:"name_servers"`
	RawWhois    string     `json:"raw_whois"`
	IPAddress   string     `json:"ip_address"`
	// SSL Certificate данные
	SSLIssuer  string     `json:"ssl_issuer"`
	SSLSubject string     `json:"ssl_subject"`
	SSLExpiry  *time.Time `json:"ssl_expiry"`
	ServerInfo string     `json:"server_info"`
	FetchedAt  time.Time  `json:"fetched_at"`
}
