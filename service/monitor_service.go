package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"whois_service/models"
	"whois_service/repository"
)

type MonitorService struct {
	statusRepo *repository.StatusRepo
	client     *http.Client
}

func NewMonitorService(sr *repository.StatusRepo) *MonitorService {
	return &MonitorService{
		statusRepo: sr,
		client: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
				DisableKeepAlives: true,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

// CheckDomain проверяет доступность домена через 3 уровня:
// 1) HTTP/HTTPS запрос — если любой ответ, значит онлайн
// 2) TCP connect к порту 443 — если соединение установлено, онлайн
// 3) DNS resolve — если IP резолвится, домен существует и скорее всего онлайн
func (s *MonitorService) CheckDomain(ctx context.Context, domainID int, domainName string) error {
	sl := &models.StatusLog{DomainID: domainID}

	// === Уровень 1: HTTP/HTTPS запрос ===
	online, httpStatus, responseTime := s.tryHTTP(ctx, domainName)

	if online {
		sl.IsOnline = true
		sl.ResponseTimeMs = responseTime
		sl.HTTPStatus = httpStatus
		log.Printf("🟢 %s ONLINE (%dms, HTTP %d)", domainName, sl.ResponseTimeMs, sl.HTTPStatus)
		return s.statusRepo.Save(sl)
	}

	// Если контекст отменён — выходим сразу
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// === Уровень 2: TCP connect (порт 443, с контекстом) ===
	start := time.Now()
	dialer := net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", domainName+":443")
	if err == nil {
		conn.Close()
		elapsed := time.Since(start)
		sl.IsOnline = true
		sl.ResponseTimeMs = int(elapsed.Milliseconds())
		sl.HTTPStatus = 0 // TCP only, no HTTP
		log.Printf("🟡 %s ONLINE via TCP (%dms)", domainName, sl.ResponseTimeMs)
		return s.statusRepo.Save(sl)
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	// === Уровень 3: DNS resolve (с контекстом) ===
	ips, err := net.DefaultResolver.LookupHost(ctx, domainName)
	if err == nil && len(ips) > 0 {
		sl.IsOnline = true
		sl.ResponseTimeMs = 0
		sl.HTTPStatus = 0
		log.Printf("🟡 %s ONLINE via DNS (IP: %s)", domainName, ips[0])
		return s.statusRepo.Save(sl)
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Ничего не помогло — домен действительно недоступен
	sl.IsOnline = false
	sl.ResponseTimeMs = 0
	sl.HTTPStatus = 0
	log.Printf("🔴 %s OFFLINE (все проверки провалились)", domainName)
	return s.statusRepo.Save(sl)
}

// tryHTTP пробует HTTPS, потом HTTP. Любой ответ (даже 403, 500) = онлайн
func (s *MonitorService) tryHTTP(ctx context.Context, domainName string) (online bool, status int, ms int) {
	// Пробуем HTTPS (с контекстом)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://%s", domainName), nil)
	if err == nil {
		start := time.Now()
		resp, err := s.client.Do(req)
		elapsed := time.Since(start)
		if err == nil {
			defer resp.Body.Close()
			return true, resp.StatusCode, int(elapsed.Milliseconds())
		}
	}

	if ctx.Err() != nil {
		return false, 0, 0
	}

	// Пробуем HTTP (с контекстом)
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s", domainName), nil)
	if err == nil {
		start := time.Now()
		resp, err := s.client.Do(req)
		elapsed := time.Since(start)
		if err == nil {
			defer resp.Body.Close()
			return true, resp.StatusCode, int(elapsed.Milliseconds())
		}
	}

	return false, 0, 0
}
