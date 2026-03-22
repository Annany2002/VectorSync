package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/Annany2002/vector-sync/internal/logger"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var (
	log = logger.NewLogger()
)

func Connect() (*sql.DB, error) {
	// Load .env if present (optional in containers where env vars are injected directly)
	_ = godotenv.Load()

	// Use DB_URL if explicitly set; otherwise build it from individual components.
	// This avoids hardcoding a connection string in .env while still supporting
	// a single DB_URL override for environments that prefer it (e.g., managed DBs).
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		host := os.Getenv("DB_HOST")
		port := os.Getenv("DB_PORT")
		user := os.Getenv("DB_USER")
		password := os.Getenv("DB_PASSWORD")
		name := os.Getenv("DB_NAME")

		if host == "" {
			host = "localhost"
		}
		if port == "" {
			port = "5432"
		}

		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, name)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}

	// Set connection pool parameters.
	// 50 open / 25 idle gives headroom for high-concurrency workloads while staying
	// well under PostgreSQL's default max_connections=100.
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Info("Database connected successfully!!!")
	return db, nil
}
