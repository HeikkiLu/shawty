package db

import (
	"database/sql"
	"urlshortener/urlshortener/internal/config"
)

func Open(cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
