package repository

import (
	"database/sql"
	"whois_service/models"
)

type WhoisRepo struct {
	db *sql.DB
}

func NewWhoisRepo(db *sql.DB) *WhoisRepo {
	return &WhoisRepo{db: db}
}

func (r *WhoisRepo) Save(record *models.WhoisRecord) error {
	_, err := r.db.Exec(
		`INSERT INTO whois_records
		 (domain_id, registrar, registrant, created_date, expiry_date,
		  name_servers, raw_whois, ip_address, ssl_issuer, ssl_subject,
		  ssl_expiry, server_info, fetched_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW())`,
		record.DomainID, record.Registrar, record.Registrant,
		record.CreatedDate, record.ExpiryDate, record.NameServers,
		record.RawWhois, record.IPAddress,
		record.SSLIssuer, record.SSLSubject, record.SSLExpiry, record.ServerInfo,
	)
	return err
}

func (r *WhoisRepo) GetLatestByDomainID(domainID int) (*models.WhoisRecord, error) {
	rec := &models.WhoisRecord{}
	err := r.db.QueryRow(
		`SELECT id, domain_id, registrar, registrant, created_date, expiry_date,
		        name_servers, raw_whois, ip_address,
		        COALESCE(ssl_issuer,''), COALESCE(ssl_subject,''),
		        ssl_expiry, COALESCE(server_info,''), fetched_at
		 FROM whois_records WHERE domain_id = $1 ORDER BY fetched_at DESC LIMIT 1`, domainID,
	).Scan(&rec.ID, &rec.DomainID, &rec.Registrar, &rec.Registrant,
		&rec.CreatedDate, &rec.ExpiryDate, &rec.NameServers,
		&rec.RawWhois, &rec.IPAddress,
		&rec.SSLIssuer, &rec.SSLSubject, &rec.SSLExpiry, &rec.ServerInfo,
		&rec.FetchedAt)
	return rec, err
}

func (r *WhoisRepo) GetAllByDomainID(domainID int) ([]models.WhoisRecord, error) {
	rows, err := r.db.Query(
		`SELECT id, domain_id, registrar, registrant, created_date, expiry_date,
		        name_servers, raw_whois, ip_address,
		        COALESCE(ssl_issuer,''), COALESCE(ssl_subject,''),
		        ssl_expiry, COALESCE(server_info,''), fetched_at
		 FROM whois_records WHERE domain_id = $1 ORDER BY fetched_at DESC`, domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]models.WhoisRecord, 0)
	for rows.Next() {
		var rec models.WhoisRecord
		if err := rows.Scan(&rec.ID, &rec.DomainID, &rec.Registrar, &rec.Registrant,
			&rec.CreatedDate, &rec.ExpiryDate, &rec.NameServers,
			&rec.RawWhois, &rec.IPAddress,
			&rec.SSLIssuer, &rec.SSLSubject, &rec.SSLExpiry, &rec.ServerInfo,
			&rec.FetchedAt); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}
