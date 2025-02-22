package repository

import "chronoverseapi/internal/entity"

type ArticleRepository struct {
	CommonRepository[entity.Article]
}
