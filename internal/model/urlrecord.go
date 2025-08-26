package model

import "time"

type URLRecord struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	LongUrl   string    `json:"long_url"`
	ShortUrl  string    `json:"short_url"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateReq struct {
	URL string `json:"url" binding:"required"`
}
