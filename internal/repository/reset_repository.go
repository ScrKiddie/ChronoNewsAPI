package repository

import (
	"chrononewsapi/internal/entity"

	"gorm.io/gorm"
)

type ResetRepository struct {
	CommonRepository[entity.Reset]
}

func NewResetRepository() *ResetRepository {
	return &ResetRepository{}
}
func (u *ResetRepository) FindByUserID(db *gorm.DB, entity *entity.Reset, userID int32) error {
	return db.Where("user_id = ?", userID).First(entity).Error
}
func (u *ResetRepository) FindByCode(db *gorm.DB, entity *entity.Reset, code string) error {
	return db.Where("code = ?", code).First(entity).Error
}
