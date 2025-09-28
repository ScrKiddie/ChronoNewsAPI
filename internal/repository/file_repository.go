package repository

import (
	"chrononewsapi/internal/entity"

	"gorm.io/gorm"
)

type FileRepository struct {
	CommonRepository[entity.File]
}

func NewFileRepository() *FileRepository {
	return &FileRepository{}
}

func (r *FileRepository) FindByID(db *gorm.DB, id int32) (*entity.File, error) {
	var file entity.File
	err := db.First(&file, id).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *FileRepository) FindAsMap(db *gorm.DB, ids []int32) map[int32]*entity.File {
	fileMap := make(map[int32]*entity.File)
	if len(ids) == 0 {
		return fileMap
	}

	var files []entity.File
	db.Where("id IN ?", ids).Find(&files)

	for i := range files {
		fileMap[int32(files[i].ID)] = &files[i]
	}

	return fileMap
}

func (r *FileRepository) LinkFilesToPost(db *gorm.DB, fileIDs []int32, postID int32) error {
	if len(fileIDs) == 0 {
		return nil
	}
	return db.Model(&entity.File{}).Where("id IN ?", fileIDs).Update("used_by_post_id", postID).Error
}

func (r *FileRepository) UnlinkUnusedFiles(db *gorm.DB, postID int32, usedFileIDs []int32) error {
	if len(usedFileIDs) == 0 {
		return db.Model(&entity.File{}).Where("used_by_post_id = ?", postID).Update("used_by_post_id", nil).Error
	}
	return db.Model(&entity.File{}).Where("used_by_post_id = ? AND id NOT IN ?", postID, usedFileIDs).Update("used_by_post_id", nil).Error
}
