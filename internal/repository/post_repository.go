package repository

import (
	"chronoverseapi/internal/entity"
	"chronoverseapi/internal/model"
	"gorm.io/gorm"
	"strings"
)

type PostRepository struct {
	CommonRepository[entity.Post]
}

func NewPostRepository() *PostRepository {
	return &PostRepository{}
}

func (r *PostRepository) Search(db *gorm.DB, request *model.PostSearch, posts *[]entity.Post) (int64, error) {
	query := db.Preload("User").Preload("Category")
	var conditions []string
	var args []interface{}

	if request.Title != "" {
		conditions = append(conditions, "LOWER(post.title) LIKE ?")
		args = append(args, "%"+strings.ToLower(request.Title)+"%")
	}

	if request.Summary != "" {
		conditions = append(conditions, "LOWER(post.summary) LIKE ?")
		args = append(args, "%"+strings.ToLower(request.Summary)+"%")
	}

	if request.CategoryName != "" {
		query = query.Joins("JOIN category ON category.id = post.category_id")
		conditions = append(conditions, "LOWER(category.name) LIKE ?")
		args = append(args, "%"+strings.ToLower(request.CategoryName)+"%")
	}

	if request.UserName != "" {
		query = query.Joins("JOIN \"user\" u ON u.id = post.user_id")
		conditions = append(conditions, "LOWER(u.name) LIKE ?")
		args = append(args, "%"+strings.ToLower(request.UserName)+"%")
	}

	if len(conditions) > 0 {
		query = query.Where(strings.Join(conditions, " OR "), args...)
	}

	if request.UserID != 0 {
		query = query.Where("post.user_id = ?", request.UserID)
	}

	var total int64
	err := query.Model(&entity.Post{}).Count(&total).Error
	if err != nil {
		return 0, err
	}

	err = query.Order("post.published_date DESC").
		Limit(int(request.Size)).
		Offset(int((request.Page - 1) * request.Size)).
		Find(posts).Error

	return total, err
}

func (r *PostRepository) FindByID(db *gorm.DB, post *entity.Post, id int32) error {
	return db.Where("id = ?", id).Preload("User").Preload("Category").First(post).Error
}

func (r *PostRepository) FindByIDAndUserID(db *gorm.DB, post *entity.Post, postID int32, userID int32) error {
	return db.Where("id = ?", postID).Where("user_id = ?", userID).Preload("User").Preload("Category").First(post).Error
}

func (r *PostRepository) Update(db *gorm.DB, post *entity.Post) error {
	return db.Model(post).
		Omit("Category", "User").
		Updates(post).Error
}

func (r *PostRepository) ExistsByUserID(db *gorm.DB, userID int32) (bool, error) {
	var exists bool
	err := db.Model(&entity.Post{}).
		Select("count(1) > 0").
		Where("user_id = ?", userID).
		Find(&exists).Error
	return exists, err
}

func (r *PostRepository) ExistsByCategoryID(db *gorm.DB, categoryID int32) (bool, error) {
	var exists bool
	err := db.Model(&entity.Post{}).
		Select("count(1) > 0").
		Where("category_id = ?", categoryID).
		Find(&exists).Error
	return exists, err
}
