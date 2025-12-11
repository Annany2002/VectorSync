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
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	db, err := sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		return nil, err
	}

	// Set connection pool parameters
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Info("Database connected successfully!!!")
	return db, nil
}