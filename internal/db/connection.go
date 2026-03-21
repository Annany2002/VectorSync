package db

import (
	"database/sql"
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

	db, err := sql.Open("postgres", os.Getenv("DB_URL"))
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
