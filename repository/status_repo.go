package repository

import (
	"database/sql"
	"whois_service/models"
)

type StatusRepo struct {
	db *sql.DB
}

func NewStatusRepo(db *sql.DB) *StatusRepo {
	return &StatusRepo{db: db}
}

func (r *StatusRepo) Save(sl *models.StatusLog) error {
	_, err := r.db.Exec(
		`INSERT INTO status_logs (domain_id, is_online, response_time_ms, http_status, checked_at)
		 VALUES ($1, $2, $3, $4, NOW())`,
		sl.DomainID, sl.IsOnline, sl.ResponseTimeMs, sl.HTTPStatus,
	)
	return err
}

func (r *StatusRepo) GetLatestByDomainID(domainID int) (*models.StatusLog, error) {
	sl := &models.StatusLog{}
	err := r.db.QueryRow(
		`SELECT id, domain_id, is_online, response_time_ms, http_status, checked_at
		 FROM status_logs WHERE domain_id = $1 ORDER BY checked_at DESC LIMIT 1`, domainID,
	).Scan(&sl.ID, &sl.DomainID, &sl.IsOnline, &sl.ResponseTimeMs, &sl.HTTPStatus, &sl.CheckedAt)
	return sl, err
}

func (r *StatusRepo) GetHistoryByDomainID(domainID int, limit int) ([]models.StatusLog, error) {
	rows, err := r.db.Query(
		`SELECT id, domain_id, is_online, response_time_ms, http_status, checked_at
		 FROM status_logs WHERE domain_id = $1 ORDER BY checked_at DESC LIMIT $2`, domainID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]models.StatusLog, 0)
	for rows.Next() {
		var sl models.StatusLog
		if err := rows.Scan(&sl.ID, &sl.DomainID, &sl.IsOnline, &sl.ResponseTimeMs, &sl.HTTPStatus, &sl.CheckedAt); err != nil {
			return nil, err
		}
		logs = append(logs, sl)
	}
	return logs, nil
}

// CountOnline считает количество доменов, которые сейчас онлайн (по последнему статусу)
func (r *StatusRepo) CountOnline() (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM (
			SELECT DISTINCT ON (domain_id) is_online
			FROM status_logs ORDER BY domain_id, checked_at DESC
		) sub WHERE is_online = true`,
	).Scan(&count)
	return count, err
}
