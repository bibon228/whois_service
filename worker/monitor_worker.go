package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"whois_service/repository"
	"whois_service/service"
)

// CheckRequest — запрос на немедленную проверку домена
type CheckRequest struct {
	DomainID   int
	DomainName string
}

type MonitorWorker struct {
	whoisSvc   *service.WhoisService
	monitorSvc *service.MonitorService
	domainRepo *repository.DomainRepo
	checkCh    chan CheckRequest // Канал для немедленных проверок
}

func NewMonitorWorker(ws *service.WhoisService, ms *service.MonitorService, dr *repository.DomainRepo) *MonitorWorker {
	return &MonitorWorker{
		whoisSvc:   ws,
		monitorSvc: ms,
		domainRepo: dr,
		checkCh:    make(chan CheckRequest, 10),
	}
}

// TriggerCheck отправляет запрос на немедленную проверку домена.
// Неблокирующий — если канал переполнен, запрос пропускается.
func (w *MonitorWorker) TriggerCheck(domainID int, domainName string) {
	select {
	case w.checkCh <- CheckRequest{DomainID: domainID, DomainName: domainName}:
		log.Printf("⚡ Запрос на проверку %s добавлен в очередь", domainName)
	default:
		log.Printf("⚠️ Очередь проверок переполнена, %s будет проверен в следующем цикле", domainName)
	}
}

// StartWhoisWorker запускает горутину, которая периодически обновляет WHOIS данные.
// Также слушает checkCh для немедленных проверок новых доменов.
func (w *MonitorWorker) StartWhoisWorker(ctx context.Context, interval time.Duration) {
	log.Println("🔄 WHOIS Worker запущен (интервал:", interval, ")")

	// Первый запуск сразу
	w.runWhoisScan(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("⏹️  WHOIS Worker остановлен")
			return
		case <-ticker.C:
			w.runWhoisScan(ctx)
		case req := <-w.checkCh:
			// Немедленная проверка нового домена
			log.Printf("⚡ Немедленная WHOIS-проверка для %s", req.DomainName)
			if err := w.whoisSvc.FetchAndSave(ctx, req.DomainID, req.DomainName); err != nil {
				log.Printf("❌ WHOIS error for %s: %v", req.DomainName, err)
			}
			// Также проверяем статус сразу
			if err := w.monitorSvc.CheckDomain(ctx, req.DomainID, req.DomainName); err != nil {
				log.Printf("❌ Status error for %s: %v", req.DomainName, err)
			}
		}
	}
}

// StartStatusWorker запускает горутину для периодической проверки доступности
func (w *MonitorWorker) StartStatusWorker(ctx context.Context, interval time.Duration) {
	log.Println("🔄 Status Worker запущен (интервал:", interval, ")")

	// Первый запуск сразу
	w.runStatusCheck(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("⏹️  Status Worker остановлен")
			return
		case <-ticker.C:
			w.runStatusCheck(ctx)
		}
	}
}

// runWhoisScan обходит все активные домены и обновляет WHOIS.
// Используем горутины с семафором для параллельного опроса (макс 3 одновременно).
// Все горутины привязаны к контексту для быстрого выхода.
func (w *MonitorWorker) runWhoisScan(ctx context.Context) {
	domains, err := w.domainRepo.GetActive()
	if err != nil {
		log.Println("❌ WHOIS scan: не удалось получить домены:", err)
		return
	}

	log.Printf("🌐 WHOIS scan: обновляем %d доменов...", len(domains))

	semaphore := make(chan struct{}, 3) // Максимум 3 параллельных WHOIS-запроса
	var wg sync.WaitGroup

	for _, d := range domains {
		// Проверяем, не отменён ли контекст
		select {
		case <-ctx.Done():
			break
		default:
		}
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		go func(id int, name string) {
			defer wg.Done()

			// Захватываем слот в семафоре, но с проверкой контекста
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}

			if err := w.whoisSvc.FetchAndSave(ctx, id, name); err != nil {
				if ctx.Err() != nil {
					return // Тихо выходим при отмене
				}
				log.Printf("❌ WHOIS error for %s: %v", name, err)
			}

			// Пауза между запросами (rate limit protection), с проверкой контекста
			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
				return
			}
		}(d.ID, d.Name)
	}

	wg.Wait()
	if ctx.Err() == nil {
		log.Println("✅ WHOIS scan завершён")
	}
}

// runStatusCheck проверяет доступность всех активных доменов параллельно
func (w *MonitorWorker) runStatusCheck(ctx context.Context) {
	domains, err := w.domainRepo.GetActive()
	if err != nil {
		log.Println("❌ Status check: не удалось получить домены:", err)
		return
	}

	log.Printf("📡 Status check: проверяем %d доменов...", len(domains))

	semaphore := make(chan struct{}, 5) // 5 параллельных проверок
	var wg sync.WaitGroup

	for _, d := range domains {
		select {
		case <-ctx.Done():
			break
		default:
		}
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		go func(id int, name string) {
			defer wg.Done()

			// Захватываем слот с проверкой контекста
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}

			if err := w.monitorSvc.CheckDomain(ctx, id, name); err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("❌ Status error for %s: %v", name, err)
			}
		}(d.ID, d.Name)
	}

	wg.Wait()
	if ctx.Err() == nil {
		log.Println("✅ Status check завершён")
	}
}
