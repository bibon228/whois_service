package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"

	"whois_service/models"
	"whois_service/repository"
)

type WhoisService struct {
	whoisRepo  *repository.WhoisRepo
	domainRepo *repository.DomainRepo
	httpClient *http.Client
}

func NewWhoisService(wr *repository.WhoisRepo, dr *repository.DomainRepo) *WhoisService {
	return &WhoisService{
		whoisRepo:  wr,
		domainRepo: dr,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // не следуем за редиректами
			},
		},
	}
}

// FetchAndSave собирает данные из 3 источников параллельно:
// 1) WHOIS lookup   2) DNS + IP resolve   3) SSL certificate + HTTP headers
func (s *WhoisService) FetchAndSave(ctx context.Context, domainID int, domainName string) error {
	record := &models.WhoisRecord{
		DomainID: domainID,
	}

	// Определяем корневой домен для WHOIS (субдомены не имеют WHOIS)
	rootDomain := extractRootDomain(domainName)

	// === 3 параллельные горутины для сбора данных ===

	type whoisResult struct {
		raw    string
		parsed *whoisparser.WhoisInfo
		err    error
	}
	type dnsResult struct {
		ip  string
		err error
	}
	type sslResult struct {
		issuer     string
		subject    string
		expiry     *time.Time
		serverInfo string
	}

	whoisCh := make(chan whoisResult, 1)
	dnsCh := make(chan dnsResult, 1)
	sslCh := make(chan sslResult, 1)

	// Горутина 1: WHOIS lookup (библиотека не поддерживает context,
	// поэтому оборачиваем в select с ctx.Done())
	go func() {
		type whoisRaw struct {
			raw string
			err error
		}
		rawCh := make(chan whoisRaw, 1)
		go func() {
			raw, err := whois.Whois(rootDomain)
			rawCh <- whoisRaw{raw: raw, err: err}
		}()

		select {
		case <-ctx.Done():
			whoisCh <- whoisResult{err: ctx.Err()}
			return
		case r := <-rawCh:
			if r.err != nil {
				whoisCh <- whoisResult{err: r.err}
				return
			}
			parsed, parseErr := whoisparser.Parse(r.raw)
			if parseErr != nil {
				whoisCh <- whoisResult{raw: r.raw, err: parseErr}
				return
			}
			whoisCh <- whoisResult{raw: r.raw, parsed: &parsed}
		}
	}()

	// Горутина 2: DNS Resolve (с контекстом)
	go func() {
		ips, err := net.DefaultResolver.LookupHost(ctx, domainName)
		if err == nil && len(ips) > 0 {
			dnsCh <- dnsResult{ip: ips[0]}
		} else {
			dnsCh <- dnsResult{err: err}
		}
	}()

	// Горутина 3: SSL Certificate + HTTP Server Header
	go func() {
		result := sslResult{}

		// Получаем SSL сертификат через TLS dial (с контекстом)
		dialer := tls.Dialer{
			NetDialer: &net.Dialer{Timeout: 10 * time.Second},
			Config:    &tls.Config{InsecureSkipVerify: true},
		}
		rawConn, err := dialer.DialContext(ctx, "tcp", domainName+":443")
		if err == nil {
			conn := rawConn.(*tls.Conn)
			defer conn.Close()
			certs := conn.ConnectionState().PeerCertificates
			if len(certs) > 0 {
				cert := certs[0]
				// Subject (кому выдан)
				if cert.Subject.Organization != nil && len(cert.Subject.Organization) > 0 {
					result.subject = strings.Join(cert.Subject.Organization, ", ")
				} else {
					result.subject = cert.Subject.CommonName
				}
				// Issuer (кем выдан)
				if cert.Issuer.Organization != nil && len(cert.Issuer.Organization) > 0 {
					result.issuer = strings.Join(cert.Issuer.Organization, ", ")
				}
				// Срок действия SSL
				expiry := cert.NotAfter
				result.expiry = &expiry
			}
		}

		// Получаем HTTP Server Header (с контекстом)
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+domainName, nil)
		if reqErr == nil {
			resp, err := s.httpClient.Do(req)
			if err == nil {
				defer resp.Body.Close()
				server := resp.Header.Get("Server")
				poweredBy := resp.Header.Get("X-Powered-By")
				if poweredBy != "" {
					server = server + " | " + poweredBy
				}
				result.serverInfo = server
			}
		}

		sslCh <- result
	}()

	// === Собираем результаты (с таймаутом 15 сек или отменой контекста) ===
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 15*time.Second)
	defer timeoutCancel()

	// DNS результат
	select {
	case dns := <-dnsCh:
		record.IPAddress = dns.ip
	case <-timeoutCtx.Done():
		log.Printf("⏱️ DNS timeout/cancelled for %s", domainName)
	}

	// WHOIS результат
	select {
	case wr := <-whoisCh:
		if wr.raw != "" {
			record.RawWhois = wr.raw
		} else if wr.err != nil {
			record.RawWhois = fmt.Sprintf("WHOIS lookup failed: %v", wr.err)
		}
		if wr.parsed != nil {
			if wr.parsed.Registrar != nil {
				record.Registrar = wr.parsed.Registrar.Name
			}
			if wr.parsed.Registrant != nil {
				record.Registrant = wr.parsed.Registrant.Name
			}
			if wr.parsed.Domain != nil {
				if wr.parsed.Domain.NameServers != nil {
					record.NameServers = strings.Join(wr.parsed.Domain.NameServers, ", ")
				}
				record.CreatedDate = parseDate(wr.parsed.Domain.CreatedDate)
				record.ExpiryDate = parseDate(wr.parsed.Domain.ExpirationDate)
			}
		}
	case <-timeoutCtx.Done():
		log.Printf("⏱️ WHOIS timeout/cancelled for %s", domainName)
		record.RawWhois = "WHOIS lookup timed out"
	}

	// SSL результат
	select {
	case ssl := <-sslCh:
		record.SSLIssuer = ssl.issuer
		record.SSLSubject = ssl.subject
		record.SSLExpiry = ssl.expiry
		record.ServerInfo = ssl.serverInfo

		// Если WHOIS не дал владельца, берём из SSL сертификата
		if record.Registrant == "" && ssl.subject != "" {
			record.Registrant = ssl.subject + " (из SSL)"
		}
	case <-timeoutCtx.Done():
		log.Printf("⏱️ SSL timeout/cancelled for %s", domainName)
	}

	// Если контекст отменён — не сохраняем частичные данные
	if ctx.Err() != nil {
		return ctx.Err()
	}

	log.Printf("📋 %s: registrar=%q, owner=%q, ssl=%q, server=%q",
		domainName, record.Registrar, record.Registrant, record.SSLIssuer, record.ServerInfo)

	return s.whoisRepo.Save(record)
}

// RefreshDomain принудительно обновляет WHOIS для домена
func (s *WhoisService) RefreshDomain(ctx context.Context, domainID int) error {
	domain, err := s.domainRepo.GetByID(domainID)
	if err != nil {
		return err
	}
	return s.FetchAndSave(ctx, domain.ID, domain.Name)
}

// extractRootDomain извлекает корневой домен из субдомена
// aws.amazon.com → amazon.com,  cloud.google.com → google.com
func extractRootDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) <= 2 {
		return domain
	}
	// Обработка доменов вроде .co.uk, .com.br
	twoPartTLDs := map[string]bool{
		"co.uk": true, "co.jp": true, "com.br": true,
		"com.au": true, "co.in": true, "org.uk": true,
	}
	if len(parts) >= 3 {
		lastTwo := parts[len(parts)-2] + "." + parts[len(parts)-1]
		if twoPartTLDs[lastTwo] {
			if len(parts) >= 4 {
				return strings.Join(parts[len(parts)-3:], ".")
			}
			return domain
		}
	}
	return strings.Join(parts[len(parts)-2:], ".")
}

// parseDate пытается распарсить дату из WHOIS ответа
func parseDate(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"02-Jan-2006",
		"January 02, 2006",
		"2006/01/02",
		"02.01.2006",
		"2006.01.02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, strings.TrimSpace(dateStr)); err == nil {
			return &t
		}
	}
	return nil
}
