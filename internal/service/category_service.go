package service

import (
	"chronoverseapi/internal/entity"
	"chronoverseapi/internal/model"
	"chronoverseapi/internal/repository"
	"chronoverseapi/internal/utility"
	"context"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
	"log/slog"
	"net/http"
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
		slog.Error(err.Error())
		return nil, utility.ErrForbidden
	}

	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrBadRequest
	}

	if categoryID, _ := s.CategoryRepository.FindIDByName(tx, request.Name); categoryID != 0 {
		return nil, utility.NewCustomError(http.StatusConflict, "Category name already exists")
	}

	category := &entity.Category{
		Name: request.Name,
	}

	if err := s.CategoryRepository.Create(tx, category); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServer
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
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
		slog.Error(err.Error())
		return nil, utility.ErrForbidden
	}

	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrBadRequest
	}

	category := new(entity.Category)
	if err := s.CategoryRepository.FindById(tx, category, request.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrNotFound
	}

	if categoryID, _ := s.CategoryRepository.FindIDByName(tx, request.Name); categoryID != 0 && categoryID != request.ID {
		return nil, utility.NewCustomError(http.StatusConflict, "Category name already exists")
	}

	category.Name = request.Name

	if err := s.CategoryRepository.Update(tx, category); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServer
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
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
		slog.Error(err.Error())
		return utility.ErrForbidden
	}

	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return utility.ErrBadRequest
	}

	ok, err := s.PostRepository.ExistsByCategoryID(tx, request.ID)
	if err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServer
	} else if ok {
		return utility.NewCustomError(http.StatusConflict, "Kategori digunakan pada berita")
	}

	category := new(entity.Category)
	if err := s.CategoryRepository.FindById(tx, category, request.ID); err != nil {
		slog.Error(err.Error())
		return utility.ErrNotFound
	}

	if err := s.CategoryRepository.Delete(tx, category); err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServer
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServer
	}

	return nil
}

func (s *CategoryService) Get(ctx context.Context, request *model.CategoryGet, auth *model.Auth) (*model.CategoryResponse, error) {
	db := s.DB.WithContext(ctx)

	if err := s.UserRepository.IsAdmin(db, auth.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrForbidden
	}

	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrBadRequest
	}

	category := new(entity.Category)
	if err := s.CategoryRepository.FindById(db, category, request.ID); err != nil {
		slog.Error(err.Error())
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
		slog.Error(err.Error())
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
