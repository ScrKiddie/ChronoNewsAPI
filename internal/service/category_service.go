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
	Validator          *validator.Validate
}

func NewCategoryService(db *gorm.DB, categoryRepository *repository.CategoryRepository, userRepository *repository.UserRepository, validator *validator.Validate) *CategoryService {
	return &CategoryService{
		DB:                 db,
		CategoryRepository: categoryRepository,
		UserRepository:     userRepository,
		Validator:          validator,
	}
}

func (c *CategoryService) Create(ctx context.Context, request *model.CategoryCreate, auth *model.Auth) (*model.CategoryResponse, error) {
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()
	if err := c.UserRepository.IsAdmin(tx, auth.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrForbidden
	}

	if err := c.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrBadRequest
	}

	if categoryID, _ := c.CategoryRepository.FindIDByName(tx, request.Name); categoryID != 0 {
		return nil, utility.NewCustomError(http.StatusConflict, "Category name already exists")
	}

	category := &entity.Category{
		Name: request.Name,
	}

	if err := c.CategoryRepository.Create(tx, category); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	return &model.CategoryResponse{
		ID:   category.ID,
		Name: category.Name,
	}, nil
}

func (c *CategoryService) Update(ctx context.Context, request *model.CategoryUpdate, auth *model.Auth) (*model.CategoryResponse, error) {
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := c.UserRepository.IsAdmin(tx, auth.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrForbidden
	}

	if err := c.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrBadRequest
	}

	category := new(entity.Category)
	if err := c.CategoryRepository.FindById(tx, category, request.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrNotFound
	}

	if categoryID, _ := c.CategoryRepository.FindIDByName(tx, request.Name); categoryID != 0 && categoryID != request.ID {
		return nil, utility.NewCustomError(http.StatusConflict, "Category name already exists")
	}

	category.Name = request.Name

	if err := c.CategoryRepository.Update(tx, category); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	return &model.CategoryResponse{
		ID:   category.ID,
		Name: category.Name,
	}, nil
}

func (c *CategoryService) Delete(ctx context.Context, request *model.CategoryDelete, auth *model.Auth) error {
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()
	if err := c.UserRepository.IsAdmin(tx, auth.ID); err != nil {
		slog.Error(err.Error())
		return utility.ErrForbidden
	}

	if err := c.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return utility.ErrBadRequest
	}

	category := new(entity.Category)
	if err := c.CategoryRepository.FindById(tx, category, request.ID); err != nil {
		slog.Error(err.Error())
		return utility.ErrNotFound
	}

	if err := c.CategoryRepository.Delete(tx, category); err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	return nil
}

func (c *CategoryService) Get(ctx context.Context, request *model.CategoryGet, auth *model.Auth) (*model.CategoryResponse, error) {
	db := c.DB.WithContext(ctx)

	if err := c.UserRepository.IsAdmin(db, auth.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrForbidden
	}

	if err := c.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrBadRequest
	}

	category := new(entity.Category)
	if err := c.CategoryRepository.FindById(db, category, request.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrNotFound
	}

	return &model.CategoryResponse{
		ID:   category.ID,
		Name: category.Name,
	}, nil
}

func (c *CategoryService) List(ctx context.Context, auth *model.Auth) (*[]model.CategoryResponse, error) {
	db := c.DB.WithContext(ctx)

	if err := c.UserRepository.IsAdmin(db, auth.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrForbidden
	}

	var categories []entity.Category
	if err := c.CategoryRepository.FindAll(db, &categories); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
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
