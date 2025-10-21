package repository

import (
	"chrononewsapi/internal/entity"

	"gorm.io/gorm"
)

type CategoryRepository struct {
	CommonRepository[entity.Category]
}

func NewCategoryRepository() *CategoryRepository {
	return &CategoryRepository{}
}

func (u *CategoryRepository) FindById(db *gorm.DB, entity *entity.Category, id int32) error {
	return db.Where("id = ?", id).First(entity).Error
}

func (u *CategoryRepository) FindAll(db *gorm.DB, categories *[]entity.Category) error {
	return db.Order("name ASC").Find(categories).Error
}

func (u *CategoryRepository) FindIDByName(db *gorm.DB, name string) (int32, error) {
	var category entity.Category
	err := db.Select("id").Where("name = ?", name).First(&category).Error
	if err != nil {
		return 0, err
	}
	return category.ID, nil
}
