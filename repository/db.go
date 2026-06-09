package repository

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// InitDB подключается к PostgreSQL, выполняет миграции и сидирует данные
func InitDB(dsn string) *sql.DB {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("❌ Не удалось подключиться к БД:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("❌ БД не отвечает:", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)

	fmt.Println("✅ Подключено к PostgreSQL!")

	runMigrations(db)
	seedData(db)

	return db
}

func runMigrations(db *sql.DB) {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(100) UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role VARCHAR(20) NOT NULL DEFAULT 'user',
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS domains (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS whois_records (
			id SERIAL PRIMARY KEY,
			domain_id INT REFERENCES domains(id) ON DELETE CASCADE,
			registrar TEXT DEFAULT '',
			registrant TEXT DEFAULT '',
			created_date TIMESTAMP,
			expiry_date TIMESTAMP,
			name_servers TEXT DEFAULT '',
			raw_whois TEXT DEFAULT '',
			ip_address VARCHAR(45) DEFAULT '',
			fetched_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS status_logs (
			id SERIAL PRIMARY KEY,
			domain_id INT REFERENCES domains(id) ON DELETE CASCADE,
			is_online BOOLEAN DEFAULT false,
			response_time_ms INT DEFAULT 0,
			http_status INT DEFAULT 0,
			checked_at TIMESTAMP DEFAULT NOW()
		)`,
		// Миграция v2: SSL-сертификат и server info
		`DO $$ BEGIN
			ALTER TABLE whois_records ADD COLUMN IF NOT EXISTS ssl_issuer TEXT DEFAULT '';
			ALTER TABLE whois_records ADD COLUMN IF NOT EXISTS ssl_subject TEXT DEFAULT '';
			ALTER TABLE whois_records ADD COLUMN IF NOT EXISTS ssl_expiry TIMESTAMP;
			ALTER TABLE whois_records ADD COLUMN IF NOT EXISTS server_info TEXT DEFAULT '';
		END $$`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			log.Fatal("❌ Миграция провалилась:", err)
		}
	}
	fmt.Println("✅ Миграции выполнены!")
}

func seedData(db *sql.DB) {
	// Создаём админа если нет пользователей
	var userCount int
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if userCount == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		db.Exec(
			"INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3)",
			"admin", string(hash), "admin",
		)
		fmt.Println("✅ Админ создан (admin / admin123)")
	}

	// Сидируем 100 популярных доменов (ON CONFLICT — идемпотентно)
	domains := []string{
		// Поисковые и технологические гиганты
		"google.com", "youtube.com", "facebook.com", "instagram.com", "twitter.com",
		"linkedin.com", "microsoft.com", "apple.com", "amazon.com", "netflix.com",
		// Соцсети и мессенджеры
		"tiktok.com", "snapchat.com", "pinterest.com", "reddit.com", "tumblr.com",
		"discord.com", "telegram.org", "whatsapp.com", "signal.org", "slack.com",
		// Разработка и DevOps
		"github.com", "gitlab.com", "bitbucket.org", "stackoverflow.com", "golang.org",
		"rust-lang.org", "python.org", "nodejs.org", "docker.com", "kubernetes.io",
		// Облачные платформы
		"aws.amazon.com", "cloud.google.com", "azure.microsoft.com", "digitalocean.com", "heroku.com",
		"vercel.com", "netlify.com", "cloudflare.com", "fastly.com", "akamai.com",
		// Новости и медиа
		"bbc.com", "cnn.com", "nytimes.com", "theguardian.com", "reuters.com",
		"forbes.com", "bloomberg.com", "techcrunch.com", "wired.com", "theverge.com",
		// Русскоязычный сегмент
		"ya.ru", "yandex.ru", "mail.ru", "vk.com", "ok.ru",
		"habr.com", "lenta.ru", "rbc.ru", "ria.ru", "tass.ru",
		// E-commerce
		"ebay.com", "alibaba.com", "shopify.com", "etsy.com", "walmart.com",
		"aliexpress.com", "wildberries.ru", "ozon.ru", "lamoda.ru", "avito.ru",
		// Стриминг и развлечения
		"spotify.com", "twitch.tv", "hulu.com", "disneyplus.com", "hbomax.com",
		"soundcloud.com", "deezer.com", "crunchyroll.com", "imdb.com", "rottentomatoes.com",
		// Образование и наука
		"wikipedia.org", "coursera.org", "udemy.com", "edx.org", "khanacademy.org",
		"medium.com", "notion.so", "figma.com", "canva.com", "dribbble.com",
		// Финансы и крипто
		"paypal.com", "stripe.com", "coinbase.com", "binance.com", "blockchain.com",
		"revolut.com", "wise.com", "robinhood.com", "kraken.com", "opensea.io",
		// Инструменты и сервисы
		"zoom.us", "dropbox.com", "evernote.com", "trello.com", "asana.com",
		"jira.atlassian.com", "1password.com", "lastpass.com", "proton.me", "duckduckgo.com",
	}

	inserted := 0
	for _, d := range domains {
		result, err := db.Exec("INSERT INTO domains (name) VALUES ($1) ON CONFLICT DO NOTHING", d)
		if err == nil {
			if rows, _ := result.RowsAffected(); rows > 0 {
				inserted++
			}
		}
	}
	if inserted > 0 {
		fmt.Printf("✅ Добавлено %d новых доменов (всего в списке: %d)\n", inserted, len(domains))
	}
}
