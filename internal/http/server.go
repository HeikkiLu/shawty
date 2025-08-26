package http

import (
	"database/sql"
	"urlshortener/urlshortener/internal/config"
	"urlshortener/urlshortener/internal/handler"
	"urlshortener/urlshortener/internal/repo"
	"urlshortener/urlshortener/internal/service"

	"github.com/gin-gonic/gin"
)

func NewServer(cfg config.Config, db *sql.DB) *gin.Engine {
	r := gin.Default()

	rp := repo.NewPostgres(db)
	sv := service.NewShortener(rp)
	h := handler.New(cfg, sv)

	r.POST("/shorten", h.Shorten)

	return r
}
