package repository

import (
	"database/sql"
	"whois_service/models"
)

type DomainRepo struct {
	db *sql.DB
}

func NewDomainRepo(db *sql.DB) *DomainRepo {
	return &DomainRepo{db: db}
}

func (r *DomainRepo) GetAll() ([]models.Domain, error) {
	rows, err := r.db.Query("SELECT id, name, is_active, created_at, updated_at FROM domains ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	domains := make([]models.Domain, 0)
	for rows.Next() {
		var d models.Domain
		if err := rows.Scan(&d.ID, &d.Name, &d.IsActive, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}
	return domains, nil
}

func (r *DomainRepo) GetActive() ([]models.Domain, error) {
	rows, err := r.db.Query("SELECT id, name, is_active, created_at, updated_at FROM domains WHERE is_active = true ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	domains := make([]models.Domain, 0)
	for rows.Next() {
		var d models.Domain
		if err := rows.Scan(&d.ID, &d.Name, &d.IsActive, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}
	return domains, nil
}

func (r *DomainRepo) GetByName(name string) (*models.Domain, error) {
	d := &models.Domain{}
	err := r.db.QueryRow(
		"SELECT id, name, is_active, created_at, updated_at FROM domains WHERE name = $1", name,
	).Scan(&d.ID, &d.Name, &d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}

func (r *DomainRepo) GetByID(id int) (*models.Domain, error) {
	d := &models.Domain{}
	err := r.db.QueryRow(
		"SELECT id, name, is_active, created_at, updated_at FROM domains WHERE id = $1", id,
	).Scan(&d.ID, &d.Name, &d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}

func (r *DomainRepo) Create(name string) (*models.Domain, error) {
	d := &models.Domain{}
	err := r.db.QueryRow(
		"INSERT INTO domains (name) VALUES ($1) RETURNING id, name, is_active, created_at, updated_at", name,
	).Scan(&d.ID, &d.Name, &d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}

func (r *DomainRepo) Update(id int, name string, isActive bool) (*models.Domain, error) {
	d := &models.Domain{}
	err := r.db.QueryRow(
		"UPDATE domains SET name = $1, is_active = $2, updated_at = NOW() WHERE id = $3 RETURNING id, name, is_active, created_at, updated_at",
		name, isActive, id,
	).Scan(&d.ID, &d.Name, &d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	return d, err
}

func (r *DomainRepo) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM domains WHERE id = $1", id)
	return err
}

func (r *DomainRepo) Count() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM domains WHERE is_active = true").Scan(&count)
	return count, err
}
