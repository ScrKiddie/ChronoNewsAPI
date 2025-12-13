package service

import (
	"chrononewsapi/internal/utility"
	"encoding/xml"
	"fmt"
	"math"
	"net/url"
	"time"

	"chrononewsapi/internal/config"
	"chrononewsapi/internal/entity"
	"chrononewsapi/internal/model"
	"chrononewsapi/internal/repository"

	"gorm.io/gorm"
)

const postsPerPage = 1000

type SitemapService struct {
	postRepository     *repository.PostRepository
	categoryRepository *repository.CategoryRepository
	cfg                *config.Config
}

func NewSitemapService(postRepository *repository.PostRepository, categoryRepository *repository.CategoryRepository, cfg *config.Config) *SitemapService {
	return &SitemapService{
		postRepository:     postRepository,
		categoryRepository: categoryRepository,
		cfg:                cfg,
	}
}

func (s *SitemapService) GenerateSitemapIndex(db *gorm.DB) ([]byte, error) {
	totalPosts, err := s.postRepository.Count(db)
	if err != nil {
		return nil, err
	}

	sitemapIndex := model.SitemapIndex{}
	sitemapIndex.Sitemaps = append(sitemapIndex.Sitemaps, model.Sitemap{
		Loc: fmt.Sprintf("%s/sitemap/categories.xml", s.cfg.Web.BaseURL),
	})

	totalPages := int(math.Ceil(float64(totalPosts) / float64(postsPerPage)))
	for i := 1; i <= totalPages; i++ {
		sitemapIndex.Sitemaps = append(sitemapIndex.Sitemaps, model.Sitemap{
			Loc: fmt.Sprintf("%s/sitemap/posts-%d.xml", s.cfg.Web.BaseURL, i),
		})
	}

	output, err := xml.MarshalIndent(sitemapIndex, "", "  ")
	if err != nil {
		return nil, err
	}

	return []byte(xml.Header + string(output)), nil
}

func (s *SitemapService) GeneratePostsSitemap(db *gorm.DB, page int) ([]byte, error) {
	offset := (page - 1) * postsPerPage
	posts, err := s.postRepository.FindAllPaged(db, postsPerPage, offset)
	if err != nil {
		return nil, err
	}

	urlset := model.URLSet{}
	for _, post := range posts {
		slug := utility.Slugify(post.Title)
		postURL := fmt.Sprintf("%s%s/%d/%s", s.cfg.Web.ClientURL, s.cfg.Web.ClientPaths.Post, post.ID, slug)
		urlset.URLs = append(urlset.URLs, model.URL{
			Loc:     postURL,
			LastMod: time.Unix(post.UpdatedAt, 0).Format(time.RFC3339),
		})
	}

	output, err := xml.MarshalIndent(urlset, "", "  ")
	if err != nil {
		return nil, err
	}

	return []byte(xml.Header + string(output)), nil
}

func (s *SitemapService) GenerateCategoriesSitemap(db *gorm.DB) ([]byte, error) {
	var categories []entity.Category
	if err := s.categoryRepository.FindAll(db, &categories); err != nil {
		return nil, err
	}

	urlset := model.URLSet{}
	for _, category := range categories {
		categoryURL := fmt.Sprintf("%s%s?category=%s", s.cfg.Web.ClientURL, s.cfg.Web.ClientPaths.Category, url.QueryEscape(category.Name))
		urlset.URLs = append(urlset.URLs, model.URL{
			Loc:     categoryURL,
			LastMod: time.Now().Format(time.RFC3339),
		})
	}

	output, err := xml.MarshalIndent(urlset, "", "  ")
	if err != nil {
		return nil, err
	}

	return []byte(xml.Header + string(output)), nil
}
