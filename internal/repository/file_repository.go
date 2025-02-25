package repository

import (
	"chronoverseapi/internal/entity"
	"gorm.io/gorm"
)

type FileRepository struct {
	CommonRepository[entity.File]
}

func NewFileRepository() *FileRepository {
	return &FileRepository{}
}

func (r *FileRepository) FindByPostId(db *gorm.DB, file *[]entity.File, postId int32) error {
	return db.Where("post_id = ?", postId).Find(file).Error
}

func (r *FileRepository) FindUnusedFile(db *gorm.DB, postId int32, keepFileNames []string) ([]entity.File, error) {
	var unusedFiles []entity.File

	if len(keepFileNames) == 0 {
		err := db.Where("post_id = ?", postId).Find(&unusedFiles).Error
		if err != nil {
			return nil, err
		}
		return unusedFiles, nil
	}

	err := db.Where("post_id = ? AND name NOT IN ?", postId, keepFileNames).Find(&unusedFiles).Error
	if err != nil {
		return nil, err
	}

	return unusedFiles, nil
}

func (r *FileRepository) DeleteUnusedFile(db *gorm.DB, postId int32, fileNames []string) error {
	if len(fileNames) == 0 {
		return nil
	}

	err := db.Where("post_id = ? AND name IN ?", postId, fileNames).Delete(&entity.File{}).Error
	if err != nil {
		return err
	}

	return nil
}
