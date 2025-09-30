package service

import (
	"chrononewsapi/internal/entity"
	"chrononewsapi/internal/model"
	"chrononewsapi/internal/repository"
	"chrononewsapi/internal/utility"
	"context"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

type CategoryService struct {
	DB                 *gorm.DB
	CategoryRepository *repository.CategoryRepository
	UserRepository     *repository.UserRepository
	PostRepository     *repository.PostRepository
	Validator          *validator.Validate
}

func NewCategoryService(db *gorm.DB, categoryRepository *repository.CategoryRepository, userRepository *repository.UserRepository, postRepository *repository.PostRepository, validator *validator.Validate) *CategoryService {
	return &CategoryService{
		DB:                 db,
		CategoryRepository: categoryRepository,
		UserRepository:     userRepository,
		PostRepository:     postRepository,
		Validator:          validator,
	}
}

func (s *CategoryService) Create(ctx context.Context, request *model.CategoryCreate, auth *model.Auth) (*model.CategoryResponse, error) {
	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()
	if err := s.UserRepository.IsAdmin(tx, auth.ID); err != nil {
		slog.Error("Failed to check admin status", "error", err)
		return nil, utility.ErrForbidden
	}

	if err := s.Validator.Struct(request); err != nil {
		slog.Error("Validation failed for category create", "error", err)
		return nil, utility.ErrBadRequest
	}

	if categoryID, _ := s.CategoryRepository.FindIDByName(tx, request.Name); categoryID != 0 {
		return nil, utility.NewCustomError(http.StatusConflict, "Category name already exists")
	}

	category := &entity.Category{
		Name: request.Name,
	}

	if err := s.CategoryRepository.Create(tx, category); err != nil {
		slog.Error("Failed to create category", "error", err)
		return nil, utility.ErrInternalServer
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error("Failed to commit transaction for category create", "error", err)
		return nil, utility.ErrInternalServer
	}

	return &model.CategoryResponse{
		ID:   category.ID,
		Name: category.Name,
	}, nil
}

func (s *CategoryService) Update(ctx context.Context, request *model.CategoryUpdate, auth *model.Auth) (*model.CategoryResponse, error) {
	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.UserRepository.IsAdmin(tx, auth.ID); err != nil {
		slog.Error("Failed to check admin status", "error", err)
		return nil, utility.ErrForbidden
	}

	if err := s.Validator.Struct(request); err != nil {
		slog.Error("Validation failed for category update", "error", err)
		return nil, utility.ErrBadRequest
	}

	category := new(entity.Category)
	if err := s.CategoryRepository.FindById(tx, category, request.ID); err != nil {
		slog.Error("Failed to find category by id", "error", err)
		return nil, utility.ErrNotFound
	}

	if categoryID, _ := s.CategoryRepository.FindIDByName(tx, request.Name); categoryID != 0 && categoryID != request.ID {
		return nil, utility.NewCustomError(http.StatusConflict, "Category name already exists")
	}

	category.Name = request.Name

	if err := s.CategoryRepository.Update(tx, category); err != nil {
		slog.Error("Failed to update category", "error", err)
		return nil, utility.ErrInternalServer
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error("Failed to commit transaction for category update", "error", err)
		return nil, utility.ErrInternalServer
	}

	return &model.CategoryResponse{
		ID:   category.ID,
		Name: category.Name,
	}, nil
}

func (s *CategoryService) Delete(ctx context.Context, request *model.CategoryDelete, auth *model.Auth) error {
	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()
	if err := s.UserRepository.IsAdmin(tx, auth.ID); err != nil {
		slog.Error("Failed to check admin status", "error", err)
		return utility.ErrForbidden
	}

	if err := s.Validator.Struct(request); err != nil {
		slog.Error("Validation failed for category delete", "error", err)
		return utility.ErrBadRequest
	}

	ok, err := s.PostRepository.ExistsByCategoryID(tx, request.ID)
	if err != nil {
		slog.Error("Failed to check if category is used by post", "error", err)
		return utility.ErrInternalServer
	} else if ok {
		return utility.NewCustomError(http.StatusConflict, "Kategori digunakan pada berita")
	}

	category := new(entity.Category)
	if err := s.CategoryRepository.FindById(tx, category, request.ID); err != nil {
		slog.Error("Failed to find category by id", "error", err)
		return utility.ErrNotFound
	}

	if err := s.CategoryRepository.Delete(tx, category); err != nil {
		slog.Error("Failed to delete category", "error", err)
		return utility.ErrInternalServer
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error("Failed to commit transaction for category delete", "error", err)
		return utility.ErrInternalServer
	}

	return nil
}

func (s *CategoryService) Get(ctx context.Context, request *model.CategoryGet, auth *model.Auth) (*model.CategoryResponse, error) {
	db := s.DB.WithContext(ctx)

	if err := s.UserRepository.IsAdmin(db, auth.ID); err != nil {
		slog.Error("Failed to check admin status", "error", err)
		return nil, utility.ErrForbidden
	}

	if err := s.Validator.Struct(request); err != nil {
		slog.Error("Validation failed for category get", "error", err)
		return nil, utility.ErrBadRequest
	}

	category := new(entity.Category)
	if err := s.CategoryRepository.FindById(db, category, request.ID); err != nil {
		slog.Error("Failed to find category by id", "error", err)
		return nil, utility.ErrNotFound
	}

	return &model.CategoryResponse{
		ID:   category.ID,
		Name: category.Name,
	}, nil
}

func (s *CategoryService) List(ctx context.Context) (*[]model.CategoryResponse, error) {
	db := s.DB.WithContext(ctx)

	var categories []entity.Category
	if err := s.CategoryRepository.FindAll(db, &categories); err != nil {
		slog.Error("Failed to find all categories", "error", err)
		return nil, utility.ErrInternalServer
	}

	if len(categories) == 0 {
		return &[]model.CategoryResponse{}, nil
	}

	var response []model.CategoryResponse
	for _, v := range categories {
		response = append(response, model.CategoryResponse{
			ID:   v.ID,
			Name: v.Name,
		})
	}

	return &response, nil
}
