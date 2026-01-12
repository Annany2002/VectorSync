package services

import (
	"database/sql"
)

// HealthService handles health check logic
type HealthService struct {
	db *sql.DB
}

// NewHealthService creates a new health service
func NewHealthService(db *sql.DB) *HealthService {
	return &HealthService{
		db: db,
	}
}

// CheckLiveness checks if the process is alive
// This should always return true if the process is running
func (s *HealthService) CheckLiveness() (bool, error) {
	return true, nil
}

// CheckReadiness checks if the service is ready to serve traffic
// This verifies database connectivity
func (s *HealthService) CheckReadiness() (bool, error) {
	if s.db == nil {
		return false, nil
	}

	// Ping the database to verify connectivity
	if err := s.db.Ping(); err != nil {
		return false, err
	}

	return true, nil
}
