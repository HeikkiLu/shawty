package handler

import (
	"mime"
	"net/http"
	"net/url"

	"urlshortener/urlshortener/internal/config"
	"urlshortener/urlshortener/internal/model"
	"urlshortener/urlshortener/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	cfg config.Config
	srv service.Shortener
}

func New(cfg config.Config, srv service.Shortener) *Handler { return &Handler{cfg: cfg, srv: srv} }

// POST /shorten
func (h *Handler) Shorten(c *gin.Context) {

	ct := c.GetHeader("Content-Type")

	mt, _, err := mime.ParseMediaType(ct)

	if err != nil || mt != "application/json" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content-Type must be application/json"})
		return
	}

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

// Get /:code -> redirect
func (h *Handler) Redirect(c *gin.Context) {
	code := c.Param("code")

	longUrl, err := h.srv.Resolve(c, code)

	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.Redirect(http.StatusFound, longUrl)
}
