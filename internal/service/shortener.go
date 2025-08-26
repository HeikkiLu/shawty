package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"urlshortener/urlshortener/internal/model"
	"urlshortener/urlshortener/internal/repo"
	"urlshortener/urlshortener/internal/util"
)

const PgUniqueViolation pq.ErrorCode = "23505"

type Shortener interface {
	Shorten(ctx context.Context, baseURL, long string) (rec model.URLRecord, created bool, err error)
	Resolve(ctx context.Context, code string) (string, error)
}

type shortener struct{ r repo.URLRepo }

func NewShortener(r repo.URLRepo) Shortener { return &shortener{r} }

func (s *shortener) Shorten(ctx context.Context, baseUrl, long string) (model.URLRecord, bool, error) {
	// Record already exists
	if rec, err := s.r.GetByLong(ctx, long); err == nil {
		return rec, false, nil
	}

	code := util.GenerateCode()
	short := baseUrl + code
	id := uuid.New().String()

	for attempt := 0; attempt < 5; attempt++ {
		rec, err := s.r.Insert(ctx, id, code, long, short)
		if err == nil {
			return rec, true, nil
		}
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == PgUniqueViolation && strings.Contains(pqErr.Detail, "(code)") {
			code = util.GenerateCode()
			short = baseUrl + code
			continue
		}
		if errors.As(err, &pqErr) && pqErr.Code == PgUniqueViolation && strings.Contains(pqErr.Detail, "(long_url)") {
			if rec, rec_err := s.r.GetByLong(ctx, long); rec_err == nil {
				return rec, false, nil
			}
		}
		return model.URLRecord{}, false, err
	}
	return model.URLRecord{}, false, errors.New("Could not allocate unique code")
}

func (s *shortener) Resolve(ctx context.Context, code string) (string, error) {
	rec, err := s.r.GetByCode(ctx, code)
	if err != nil {
		return "", err
	}

	return rec.LongUrl, nil
}
