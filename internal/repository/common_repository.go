package repository

import "gorm.io/gorm"

type CommonRepository[T any] struct {
}

func (c *CommonRepository[T]) Create(db *gorm.DB, entity *T) error {
	return db.Create(entity).Error
}
func (c *CommonRepository[T]) Update(db *gorm.DB, entity *T) error {
	return db.Save(entity).Error
}
func (c *CommonRepository[T]) Delete(db *gorm.DB, entity *T) error {
	return db.Delete(entity).Error
}
