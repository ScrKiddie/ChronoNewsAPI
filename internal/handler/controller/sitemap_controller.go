package controller

import (
	"chrononewsapi/internal/service"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type SitemapController struct {
	sitemapService *service.SitemapService
	DB             *gorm.DB
}

func NewSitemapController(sitemapService *service.SitemapService, db *gorm.DB) *SitemapController {
	return &SitemapController{
		sitemapService: sitemapService,
		DB:             db,
	}
}

func (c *SitemapController) GetSitemapIndex(w http.ResponseWriter, _ *http.Request) {
	sitemap, err := c.sitemapService.GenerateSitemapIndex(c.DB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	if _, err := w.Write(sitemap); err != nil {
		slog.Error("Failed to write sitemap index", "error", err)
		http.Error(w, "Failed to write sitemap", http.StatusInternalServerError)
	}
}

func (c *SitemapController) GetPostsSitemap(w http.ResponseWriter, r *http.Request) {
	pageStr := chi.URLParam(r, "page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		http.Error(w, "Invalid page number", http.StatusBadRequest)
		return
	}

	sitemap, err := c.sitemapService.GeneratePostsSitemap(c.DB, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	if _, err := w.Write(sitemap); err != nil {
		slog.Error("Failed to write posts sitemap", "error", err)
		http.Error(w, "Failed to write sitemap", http.StatusInternalServerError)
	}
}

func (c *SitemapController) GetCategoriesSitemap(w http.ResponseWriter, _ *http.Request) {
	sitemap, err := c.sitemapService.GenerateCategoriesSitemap(c.DB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	if _, err := w.Write(sitemap); err != nil {
		slog.Error("Failed to write categories sitemap", "error", err)
		http.Error(w, "Failed to write sitemap", http.StatusInternalServerError)
	}
}
