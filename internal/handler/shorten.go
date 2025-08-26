package handler

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"urlshortener/urlshortener/internal/config"
	"urlshortener/urlshortener/internal/model"
	"urlshortener/urlshortener/internal/service"
)

type Handler struct {
	cfg config.Config
	srv service.Shortener
}

func New(cfg config.Config, srv service.Shortener) *Handler { return &Handler{cfg: cfg, srv: srv} }

// POST /shorten
func (h *Handler) Shorten(c *gin.Context) {
	var req model.CreateReq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing field: url"})
		return
	}

	parsedUrl, err := url.ParseRequestURI(req.URL)
	if err != nil || (parsedUrl.Scheme != "http" && parsedUrl.Scheme != "https") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Malformed or unsupported URL"})
		return
	}

	rec, created, err := h.srv.Shorten(c.Request.Context(), h.cfg.BaseURL, parsedUrl.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if created {
		c.IndentedJSON(http.StatusCreated, rec)
	} else {
		c.IndentedJSON(http.StatusOK, rec)
	}
}
