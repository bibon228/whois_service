package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"

	"whois_service/handlers"
	"whois_service/middleware"
	"whois_service/repository"
	"whois_service/service"
	"whois_service/worker"
)

func main() {
	
	db := repository.InitDB(AppConfig.DSN)
	defer db.Close()


	userRepo := repository.NewUserRepo(db)
	domainRepo := repository.NewDomainRepo(db)
	whoisRepo := repository.NewWhoisRepo(db)
	statusRepo := repository.NewStatusRepo(db)

	authSvc := service.NewAuthService(userRepo, AppConfig.JWTSecret)
	domainSvc := service.NewDomainService(domainRepo, whoisRepo, statusRepo)
	whoisSvc := service.NewWhoisService(whoisRepo, domainRepo)
	monitorSvc := service.NewMonitorService(statusRepo)

	// Worker создаём ДО handler'ов, чтобы передать в AdminHandler
	mw := worker.NewMonitorWorker(whoisSvc, monitorSvc, domainRepo)

	authHandler := handlers.NewAuthHandler(authSvc)
	domainHandler := handlers.NewDomainHandler(domainSvc)
	adminHandler := handlers.NewAdminHandler(domainSvc, whoisSvc, mw)
	pageHandler := handlers.NewPageHandler("static")

	authMw := middleware.NewAuthMiddleware(AppConfig.JWTSecret)


	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)

	fileServer := http.FileServer(http.Dir("static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))


	r.Get("/", pageHandler.Index)
	r.Get("/domain/{name}", pageHandler.DomainPage)
	r.Get("/login", pageHandler.LoginPage)
	r.Get("/register", pageHandler.RegisterPage)
	r.Get("/admin", pageHandler.AdminPage)

	
	r.Get("/api/domains", domainHandler.List)
	r.Get("/api/domains/{name}", domainHandler.Get)
	r.Get("/api/domains/{name}/status", domainHandler.StatusHistory)
	r.Get("/api/domains/{name}/whois", domainHandler.Whois)


	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(5, 1*time.Minute))
		r.Post("/api/login", authHandler.Login)
		r.Post("/api/register", authHandler.Register)
	})

	r.Route("/api/admin", func(r chi.Router) {
		r.Use(authMw.RequireAuth)
		r.Use(authMw.RequireAdmin)
		r.Get("/stats", adminHandler.Stats)
		r.Post("/domains", adminHandler.CreateDomain)
		r.Put("/domains/{id}", adminHandler.UpdateDomain)
		r.Delete("/domains/{id}", adminHandler.DeleteDomain)
		r.Post("/domains/{id}/refresh", adminHandler.RefreshDomain)
	})

	
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		mw.StartWhoisWorker(ctx, 1*time.Hour)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		mw.StartStatusWorker(ctx, 5*time.Minute)
	}()

	
	srv := &http.Server{
		Addr:         AppConfig.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		fmt.Printf("\n⚡ Получен сигнал: %v. Завершаем работу...\n", sig)

		cancel() // Отменяем контекст — все воркеры и сетевые вызовы прерываются

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		srv.Shutdown(shutdownCtx) 
	}()

	fmt.Println("══════════════════════════════════════════════")
	fmt.Println("  🌐 WHOIS Service запущен")
	fmt.Printf("  🔗 http://localhost%s\n", AppConfig.Port)
	fmt.Println("  👤 Админ: admin / admin123")
	fmt.Println("  🛡️  Rate Limit: 5 req/min на /api/login, /api/register")
	fmt.Println("══════════════════════════════════════════════")

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal("❌ Сервер упал:", err)
	}

	// Ждём завершения воркеров с дедлайном 10 секунд
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("👋 Сервер остановлен. До свидания!")
	case <-time.After(10 * time.Second):
		fmt.Println("⚠️ Воркеры не завершились за 10 секунд. Принудительный выход.")
	}
}
