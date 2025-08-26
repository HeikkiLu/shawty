package repo

import (
	"context"
	"database/sql"

	"urlshortener/urlshortener/internal/model"
)

type URLRepo interface {
	GetByLong(ctx context.Context, long string) (model.URLRecord, error)
	GetByCode(ctx context.Context, code string) (model.URLRecord, error)
	Insert(ctx context.Context, id string, code string, long string, short string) (model.URLRecord, error)
}

type PostgresRepo struct{ db *sql.DB }

func NewPostgres(db *sql.DB) *PostgresRepo { return &PostgresRepo{db} }

func (r *PostgresRepo) GetByLong(ctx context.Context, long string) (model.URLRecord, error) {
	const q = `SELECT id, code, long_url, short_url, created_at FROM url_records WHERE long_url=$1`

	var rec model.URLRecord
	err := r.db.QueryRowContext(ctx, q, long).Scan(&rec.ID, &rec.Code, &rec.LongUrl, &rec.ShortUrl, &rec.CreatedAt)

	return rec, err
}

func (r *PostgresRepo) GetByCode(ctx context.Context, code string) (model.URLRecord, error) {
	const q = `SELECT id, code, long_url, short_url, created_at FROM url_records WHERE code=$1`
	var rec model.URLRecord
	err := r.db.QueryRowContext(ctx, q, code).Scan(&rec.ID, &rec.Code, &rec.LongUrl, &rec.ShortUrl, &rec.CreatedAt)
	return rec, err
}

func (r *PostgresRepo) Insert(ctx context.Context, id string, code string, long string, short string) (model.URLRecord, error) {
	const q = `
		INSERT INTO url_records (id, code, long_url, short_url)
		VALUES ($1, $2, $3, $4)
		RETURNING id, code, long_url, short_url, created_at`

	var rec model.URLRecord

	err := r.db.QueryRowContext(ctx, q, id, code, long, short).
		Scan(&rec.ID, &rec.Code, &rec.LongUrl, &rec.ShortUrl, &rec.CreatedAt)

	return rec, err
}
