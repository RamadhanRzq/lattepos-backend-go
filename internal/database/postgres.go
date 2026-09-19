package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"github.com/ramadhanrzq/backend-go/internal/config"
)

// Open membuka koneksi PostgreSQL dan memastikan database dapat dijangkau.
func Open(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("gagal membuka koneksi database: %w", err)
	}

	// Connection pool configuration.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf(
			"database tidak dapat dijangkau (%s:%s/%s): %w",
			cfg.DB.Host,
			cfg.DB.Port,
			cfg.DB.Name,
			err,
		)
	}

	return db, nil
}
