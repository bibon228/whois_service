package service

import (
	"time"
	"whois_service/models"
	"whois_service/repository"
)

type DomainService struct {
	domainRepo *repository.DomainRepo
	whoisRepo  *repository.WhoisRepo
	statusRepo *repository.StatusRepo
}

func NewDomainService(dr *repository.DomainRepo, wr *repository.WhoisRepo, sr *repository.StatusRepo) *DomainService {
	return &DomainService{domainRepo: dr, whoisRepo: wr, statusRepo: sr}
}

// ListDomains возвращает все домены, обогащённые последним статусом и WHOIS
func (s *DomainService) ListDomains() ([]models.DomainWithStatus, error) {
	domains, err := s.domainRepo.GetAll()
	if err != nil {
		return nil, err
	}

	result := make([]models.DomainWithStatus, 0, len(domains))
	for _, d := range domains {
		dws := models.DomainWithStatus{Domain: d}

		if status, err := s.statusRepo.GetLatestByDomainID(d.ID); err == nil {
			dws.IsOnline = &status.IsOnline
			dws.ResponseTimeMs = &status.ResponseTimeMs
		}

		if whois, err := s.whoisRepo.GetLatestByDomainID(d.ID); err == nil {
			dws.IPAddress = whois.IPAddress
			dws.Registrar = whois.Registrar
			dws.ExpiryDate = whois.ExpiryDate
		}

		result = append(result, dws)
	}

	return result, nil
}

func (s *DomainService) GetDomain(name string) (*models.DomainWithStatus, error) {
	d, err := s.domainRepo.GetByName(name)
	if err != nil {
		return nil, err
	}

	dws := &models.DomainWithStatus{Domain: *d}

	if status, err := s.statusRepo.GetLatestByDomainID(d.ID); err == nil {
		dws.IsOnline = &status.IsOnline
		dws.ResponseTimeMs = &status.ResponseTimeMs
	}

	if whois, err := s.whoisRepo.GetLatestByDomainID(d.ID); err == nil {
		dws.IPAddress = whois.IPAddress
		dws.Registrar = whois.Registrar
		dws.ExpiryDate = whois.ExpiryDate
	}

	return dws, nil
}

func (s *DomainService) GetStatusHistory(name string, limit int) ([]models.StatusLog, error) {
	d, err := s.domainRepo.GetByName(name)
	if err != nil {
		return nil, err
	}
	return s.statusRepo.GetHistoryByDomainID(d.ID, limit)
}

func (s *DomainService) GetWhois(name string) (*models.WhoisRecord, error) {
	d, err := s.domainRepo.GetByName(name)
	if err != nil {
		return nil, err
	}
	return s.whoisRepo.GetLatestByDomainID(d.ID)
}

func (s *DomainService) CreateDomain(name string) (*models.Domain, error) {
	return s.domainRepo.Create(name)
}

func (s *DomainService) UpdateDomain(id int, name string, isActive bool) (*models.Domain, error) {
	return s.domainRepo.Update(id, name, isActive)
}

func (s *DomainService) DeleteDomain(id int) error {
	return s.domainRepo.Delete(id)
}

// GetStats собирает статистику для админ-дашборда
func (s *DomainService) GetStats() (*models.DashboardStats, error) {
	total, err := s.domainRepo.Count()
	if err != nil {
		return nil, err
	}

	online, _ := s.statusRepo.CountOnline()

	// Находим домены, у которых скоро истекает срок регистрации
	allDomains, _ := s.ListDomains()
	expiring := make([]models.DomainWithStatus, 0)
	for _, d := range allDomains {
		if d.ExpiryDate != nil {
			daysLeft := int(time.Until(*d.ExpiryDate).Hours() / 24)
			if daysLeft <= 30 && daysLeft > 0 {
				expiring = append(expiring, d)
			}
		}
	}

	return &models.DashboardStats{
		TotalDomains:   total,
		OnlineDomains:  online,
		OfflineDomains: total - online,
		ExpiringSoon:   expiring,
	}, nil
}
