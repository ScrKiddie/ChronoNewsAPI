package service

import (
	"chrononewsapi/internal/adapter"
	"chrononewsapi/internal/constant"
	"chrononewsapi/internal/entity"
	"chrononewsapi/internal/model"
	"chrononewsapi/internal/repository"
	"chrononewsapi/internal/utility"
	"context"
	"log/slog"
	"mime/multipart"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type FileService struct {
	DB             *gorm.DB
	FileRepository *repository.FileRepository
	StorageAdapter *adapter.StorageAdapter
	Config         *viper.Viper
	Validator      *validator.Validate
}

func NewFileService(db *gorm.DB, fileRepository *repository.FileRepository, storageAdapter *adapter.StorageAdapter, config *viper.Viper, validator *validator.Validate) *FileService {
	return &FileService{
		DB:             db,
		FileRepository: fileRepository,
		StorageAdapter: storageAdapter,
		Config:         config,
		Validator:      validator,
	}
}

func (s *FileService) UploadImage(ctx context.Context, fileHeader *multipart.FileHeader) (*model.ImageUploadResponse, error) {
	uploadValidation := &model.FileUpload{File: fileHeader}
	if err := s.Validator.Struct(uploadValidation); err != nil {
		slog.Error("Validation failed for image upload", "error", err)
		return nil, utility.ErrBadRequest
	}

	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	fileName := utility.CreateFileName(fileHeader)

	fileEntity := entity.File{
		Name:   fileName,
		Status: constant.FileStatusPending,
	}

	if err := s.FileRepository.Create(tx, &fileEntity); err != nil {
		slog.Error("Failed to create file record in database", "error", err)
		return nil, utility.ErrInternalServer
	}

	if err := s.StorageAdapter.Store(fileHeader, s.Config.GetString("storage.post")+fileName); err != nil {
		slog.Error("Failed to store file to storage", "error", err)
		return nil, utility.ErrInternalServer
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error("Failed to commit transaction for file upload", "error", err)
		return nil, utility.ErrInternalServer
	}

	return &model.ImageUploadResponse{
		ID:   int32(fileEntity.ID),
		Name: fileEntity.Name,
	}, nil
}
