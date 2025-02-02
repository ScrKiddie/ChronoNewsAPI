package repository

import (
	"ChronoverseAPI/internal/constant"
	"ChronoverseAPI/internal/entity"
	"ChronoverseAPI/internal/model"
	"gorm.io/gorm"
	"strings"
)

type UserRepository struct {
	CommonRepository[entity.User]
}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

func (u *UserRepository) FindIDByEmail(db *gorm.DB, email string) int32 {
	var user entity.User
	if err := db.Select("id").Where("email = ?", email).First(&user).Error; err != nil {
		return 0
	}
	return user.ID
}

func (u *UserRepository) FindIDByPhoneNumber(db *gorm.DB, phoneNumber string) int32 {
	var user entity.User
	if err := db.Select("id").Where("phone_number = ?", phoneNumber).First(&user).Error; err != nil {
		return 0
	}
	return user.ID
}

func (u *UserRepository) FindPasswordByEmail(db *gorm.DB, entity *entity.User, email string) error {
	return db.Where("email = ?", email).First(entity).Error
}

func (u *UserRepository) FindPasswordByPhoneNumber(db *gorm.DB, entity *entity.User, phoneNumber string) error {
	return db.Where("phone_number = ?", phoneNumber).First(entity).Error
}

func (u *UserRepository) FindById(db *gorm.DB, entity *entity.User, id int32) error {
	return db.Where("id = ?", id).First(entity).Error
}

func (u *UserRepository) FindByIds(db *gorm.DB, entity *entity.User, ids []int32) error {
	return db.Where("id IN ?", ids).Find(&entity).Error
}

func (u *UserRepository) IsAdmin(db *gorm.DB, id int32) error {
	return db.Where("id = ?", id).Where("role = ?", constant.Admin).First(&entity.User{}).Error
}

func (u *UserRepository) Search(db *gorm.DB, request *model.UserSearch, entities *[]entity.User) (int64, error) {
	var conditions []string
	var args []interface{}

	if request.Email != "" {
		conditions = append(conditions, "LOWER(email) LIKE ?")
		args = append(args, "%"+strings.ToLower(request.Email)+"%")
	}

	if request.PhoneNumber != "" {
		conditions = append(conditions, "phone_number LIKE ?")
		args = append(args, "%"+request.PhoneNumber+"%")
	}

	if request.Name != "" {
		conditions = append(conditions, "LOWER(name) LIKE ?")
		args = append(args, "%"+strings.ToLower(request.Name)+"%")
	}

	if len(conditions) > 0 {
		db = db.Where(strings.Join(conditions, " OR "), args...)
	}

	if request.Role != "" {
		db = db.Where("role = ?", request.Role)
	}

	var total int64
	err := db.Model(&entity.User{}).Count(&total).Error
	if err != nil {
		return 0, err
	}

	err = db.Order("name ASC").
		Limit(int(request.Size)).
		Offset(int((request.Page - 1) * request.Size)).
		Find(entities).
		Error

	return total, err
}
