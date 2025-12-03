package service

import (
	"chrononewsapi/internal/adapter"
	"chrononewsapi/internal/config"
	"chrononewsapi/internal/constant"
	"chrononewsapi/internal/entity"
	"chrononewsapi/internal/model"
	"chrononewsapi/internal/repository"
	"chrononewsapi/internal/utility"
	"context"
	"log/slog"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

type PostService struct {
	DB                 *gorm.DB
	PostRepository     *repository.PostRepository
	UserRepository     *repository.UserRepository
	FileRepository     *repository.FileRepository
	CategoryRepository *repository.CategoryRepository
	StorageAdapter     *adapter.StorageAdapter
	Validator          *validator.Validate
	Config             *config.Config
}

func NewPostService(
	db *gorm.DB,
	postRepository *repository.PostRepository,
	userRepository *repository.UserRepository,
	fileRepository *repository.FileRepository,
	categoryRepository *repository.CategoryRepository,
	storageAdapter *adapter.StorageAdapter,
	validator *validator.Validate,
	config *config.Config,
) *PostService {
	return &PostService{
		DB:                 db,
		PostRepository:     postRepository,
		UserRepository:     userRepository,
		FileRepository:     fileRepository,
		CategoryRepository: categoryRepository,
		StorageAdapter:     storageAdapter,
		Validator:          validator,
		Config:             config,
	}
}

func (s *PostService) Search(ctx context.Context, request *model.PostSearch) (*[]model.PostResponseWithPreload, *model.Pagination, error) {
	db := s.DB.WithContext(ctx)

	var excludeIDs []uint
	if request.ExcludeIDs != "" {
		idStrs := strings.Split(request.ExcludeIDs, ",")
		for _, idStr := range idStrs {
			id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32)
			if err != nil {
				slog.Error("Failed to parse excludeIds", "idStr", idStr, "error", err)
				return nil, nil, utility.ErrBadRequest
			}
			excludeIDs = append(excludeIDs, uint(id))
		}
	}

	var posts []entity.Post
	total, err := s.PostRepository.Search(db, request, &posts, excludeIDs)
	if err != nil {
		slog.Error("Failed to search posts", "error", err)
		return nil, nil, utility.ErrInternalServer
	}

	if len(posts) == 0 {
		return &[]model.PostResponseWithPreload{}, &model.Pagination{}, nil
	}

	var response []model.PostResponseWithPreload
	for _, post := range posts {
		var thumbnail string
		if len(post.Files) > 0 {
			thumbnail = post.Files[0].Name
		}

		response = append(response, model.PostResponseWithPreload{
			ID:        post.ID,
			Title:     post.Title,
			Summary:   post.Summary,
			CreatedAt: post.CreatedAt,
			UpdatedAt: post.UpdatedAt,
			Thumbnail: thumbnail,
			ViewCount: post.ViewCount,
			User: &model.UserResponse{
				ID:             post.User.ID,
				Name:           post.User.Name,
				ProfilePicture: post.User.ProfilePicture,
				PhoneNumber:    post.User.PhoneNumber,
				Email:          post.User.Email,
				Role:           post.User.Role,
			},
			Category: &model.CategoryResponse{
				ID:   post.Category.ID,
				Name: post.Category.Name,
			},
		})
	}

	pagination := model.Pagination{
		TotalItem: total,
	}

	if request.Page != 0 && request.Size != 0 {
		pagination.Page = request.Page
		pagination.Size = request.Size
		pagination.TotalPage = int64(math.Ceil(float64(total) / float64(request.Size)))
	} else {
		pagination.TotalPage = 1
	}

	return &response, &pagination, nil
}

func (s *PostService) Get(ctx context.Context, request *model.PostGet) (*model.PostResponseWithPreload, error) {
	if err := s.Validator.Struct(request); err != nil {
		slog.Error("Validation failed for post get", "error", err)
		return nil, utility.ErrBadRequest
	}

	db := s.DB.WithContext(ctx)

	post := &entity.Post{}
	if err := s.PostRepository.FindByID(db, post, request.ID); err != nil {
		slog.Error("Failed to find post by ID", "error", err)
		return nil, utility.ErrNotFound
	}

	fileIDs, err := utility.ExtractFileIDsFromContent(post.Content)
	if err != nil {
		slog.Error("Failed to extract file IDs from content", "error", err)
		return nil, utility.ErrInternalServer
	}
	fileMap := s.FileRepository.FindAsMap(db, fileIDs)

	rebuiltContent, err := utility.RebuildContentWithImageSrc(post.Content, fileMap)
	if err != nil {
		slog.Error("Failed to rebuild content with image src", "error", err)
		return nil, utility.ErrInternalServer
	}

	var thumbnail string
	for _, file := range post.Files {
		if file.Type == constant.FileTypeThumbnail {
			thumbnail = file.Name
			break
		}
	}

	response := &model.PostResponseWithPreload{
		ID:        post.ID,
		Title:     post.Title,
		Summary:   post.Summary,
		Content:   rebuiltContent,
		ViewCount: post.ViewCount,
		Thumbnail: thumbnail,
		User: &model.UserResponse{
			ID:             post.User.ID,
			Name:           post.User.Name,
			ProfilePicture: post.User.ProfilePicture,
			PhoneNumber:    post.User.PhoneNumber,
			Email:          post.User.Email,
			Role:           post.User.Role,
		},
		Category: &model.CategoryResponse{
			ID:   post.Category.ID,
			Name: post.Category.Name,
		},
		CreatedAt: post.CreatedAt,
		UpdatedAt: post.UpdatedAt,
	}

	return response, nil
}

func (s *PostService) IncrementViewCount(ctx context.Context, request *model.PostIncrementView) error {
	if err := s.Validator.Struct(request); err != nil {
		slog.Error("Validation failed for post increment view", "error", err)
		return utility.ErrBadRequest
	}

	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	post := &entity.Post{}
	if err := s.PostRepository.FindByID(tx, post, request.ID); err != nil {
		slog.Error("Failed to find post by ID for incrementing view", "error", err)
		return utility.ErrNotFound
	}

	post.ViewCount = post.ViewCount + 1
	if err := s.PostRepository.Update(tx, post); err != nil {
		slog.Error("Failed to update post view count", "error", err)
		return utility.ErrInternalServer
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error("Failed to commit transaction for incrementing view count", "error", err)
		return utility.ErrInternalServer
	}

	return nil
}

func (s *PostService) Create(ctx context.Context, request *model.PostCreate, auth *model.Auth) (*model.PostResponse, error) {
	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.UserRepository.IsAdmin(tx, auth.ID); err != nil {
		request.UserID = auth.ID
	}
	if request.UserID == 0 {
		request.UserID = auth.ID
	}

	if err := s.Validator.Struct(request); err != nil {
		slog.Error("Validation failed for post create", "error", err)
		return nil, utility.ErrBadRequest
	}

	if err := s.UserRepository.FindByID(tx, &entity.User{}, request.UserID); err != nil {
		slog.Error("User not found for post create", "error", err)
		return nil, utility.ErrNotFound
	}

	if err := s.CategoryRepository.FindById(tx, &entity.Category{}, request.CategoryID); err != nil {
		slog.Error("Category not found for post create", "error", err)
		return nil, utility.ErrNotFound
	}

	fileIDs, err := utility.ExtractFileIDsFromContent(request.Content)
	if err != nil {
		slog.Error("Failed to parse content for file IDs", "error", err)
		return nil, utility.ErrInternalServer
	}

	sanitizedContent, err := utility.StripImageSrcFromContent(request.Content)
	if err != nil {
		slog.Error("Failed to strip image src from content", "error", err)
		return nil, utility.ErrInternalServer
	}

	post := &entity.Post{
		Title:      request.Title,
		Summary:    request.Summary,
		Content:    sanitizedContent,
		UserID:     request.UserID,
		CategoryID: request.CategoryID,
	}

	if err := s.PostRepository.Create(tx, post); err != nil {
		slog.Error("Failed to create post", "error", err)
		return nil, utility.ErrInternalServer
	}

	var thumbnailName string
	if request.Thumbnail != nil {
		thumbnailName = utility.CreateFileName(request.Thumbnail)
		thumbnailFile := &entity.File{
			Name:         thumbnailName,
			Type:         constant.FileTypeThumbnail,
			UsedByPostID: &post.ID,
		}
		if err := s.FileRepository.Create(tx, thumbnailFile); err != nil {
			slog.Error("Failed to create thumbnail file record", "error", err)
			return nil, utility.ErrInternalServer
		}
	}

	if err := s.FileRepository.LinkFilesToPost(tx, fileIDs, post.ID); err != nil {
		slog.Error("Failed to link files to post", "error", err)
		return nil, utility.ErrInternalServer
	}

	if request.Thumbnail != nil {
		storagePath := s.Config.Storage.Post
		fullPath := filepath.Join(storagePath, thumbnailName)

		if err := s.StorageAdapter.Store(request.Thumbnail, fullPath); err != nil {
			slog.Error("Failed to store thumbnail file", "error", err)
			return nil, utility.ErrInternalServer
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error("Failed to commit transaction for post create", "error", err)
		return nil, utility.ErrInternalServer
	}

	response := &model.PostResponse{
		ID:         post.ID,
		CategoryID: post.CategoryID,
		UserID:     post.UserID,
		Title:      post.Title,
		Summary:    post.Summary,
		Content:    post.Content,
		CreatedAt:  post.CreatedAt,
		UpdatedAt:  post.UpdatedAt,
		Thumbnail:  thumbnailName,
	}

	return response, nil
}

func (s *PostService) Update(ctx context.Context, request *model.PostUpdate, auth *model.Auth) (*model.PostResponse, error) {
	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	post := &entity.Post{}
	if err := s.UserRepository.IsAdmin(tx, auth.ID); err != nil {
		request.UserID = auth.ID
		if err := s.PostRepository.FindByIDAndUserID(tx, post, request.ID, auth.ID); err != nil {
			slog.Error("Failed to find post by ID and UserID for update", "error", err)
			return nil, utility.ErrNotFound
		}
	} else {
		if err := s.PostRepository.FindByID(tx, post, request.ID); err != nil {
			slog.Error("Failed to find post by ID for update", "error", err)
			return nil, utility.ErrNotFound
		}
	}

	if err := s.Validator.Struct(request); err != nil {
		slog.Error("Validation failed for post update", "error", err)
		return nil, utility.ErrBadRequest
	}

	if err := s.CategoryRepository.FindById(tx, &entity.Category{}, request.CategoryID); err != nil {
		slog.Error("Category not found for post update", "error", err)
		return nil, utility.ErrNotFound
	}

	currentFileIDs, err := utility.ExtractFileIDsFromContent(request.Content)
	if err != nil {
		slog.Error("Failed to parse content for file IDs", "error", err)
		return nil, utility.ErrInternalServer
	}

	sanitizedContent, err := utility.StripImageSrcFromContent(request.Content)
	if err != nil {
		slog.Error("Failed to strip image src from content", "error", err)
		return nil, utility.ErrInternalServer
	}

	var oldThumbnailFile *entity.File
	for i := range post.Files {
		if post.Files[i].Type == constant.FileTypeThumbnail {
			oldThumbnailFile = &post.Files[i]
			break
		}
	}

	var newThumbnailName string
	var newThumbnailFile *entity.File

	if request.Thumbnail != nil {
		newThumbnailName = utility.CreateFileName(request.Thumbnail)
		newThumbnailFile = &entity.File{
			Name: newThumbnailName,
			Type: constant.FileTypeThumbnail,
		}
		if err := s.FileRepository.Create(tx, newThumbnailFile); err != nil {
			slog.Error("Failed to create new thumbnail file record", "error", err)
			return nil, utility.ErrInternalServer
		}
		currentFileIDs = append(currentFileIDs, newThumbnailFile.ID)
	}

	if request.Thumbnail == nil && !request.DeleteThumbnail && oldThumbnailFile != nil {
		currentFileIDs = append(currentFileIDs, oldThumbnailFile.ID)
	}

	post.Title = request.Title
	post.Summary = request.Summary
	post.Content = sanitizedContent
	post.CategoryID = request.CategoryID

	if request.UserID != 0 {
		if err := s.UserRepository.FindByID(tx, &entity.User{}, request.UserID); err != nil {
			slog.Error("User not found for post update", "error", err)
			return nil, utility.ErrNotFound
		}
		post.UserID = request.UserID
	} else {
		post.UserID = auth.ID
	}

	if err := s.PostRepository.Update(tx, post); err != nil {
		slog.Error("Failed to update post", "error", err)
		return nil, utility.ErrInternalServer
	}

	if err := s.FileRepository.UnlinkUnusedFiles(tx, post.ID, currentFileIDs); err != nil {
		slog.Error("Failed to unlink unused files", "error", err)
		return nil, utility.ErrInternalServer
	}

	if err := s.FileRepository.LinkFilesToPost(tx, currentFileIDs, post.ID); err != nil {
		slog.Error("Failed to link files to post", "error", err)
		return nil, utility.ErrInternalServer
	}

	if request.Thumbnail != nil {
		storagePath := s.Config.Storage.Post
		fullPath := filepath.Join(storagePath, newThumbnailName)

		if err := s.StorageAdapter.Store(request.Thumbnail, fullPath); err != nil {
			slog.Error("Failed to store new thumbnail file", "error", err)
			return nil, utility.ErrInternalServer
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error("Failed to commit transaction for post update", "error", err)
		return nil, utility.ErrInternalServer
	}

	if newThumbnailName == "" && oldThumbnailFile != nil && !request.DeleteThumbnail {
		newThumbnailName = oldThumbnailFile.Name
	}

	response := &model.PostResponse{
		ID:         post.ID,
		CategoryID: post.CategoryID,
		UserID:     post.UserID,
		Title:      post.Title,
		Summary:    post.Summary,
		Content:    post.Content,
		CreatedAt:  post.CreatedAt,
		UpdatedAt:  post.UpdatedAt,
		Thumbnail:  newThumbnailName,
	}

	return response, nil
}

func (s *PostService) Delete(ctx context.Context, request *model.PostDelete, auth *model.Auth) error {
	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.Validator.Struct(request); err != nil {
		slog.Error("Validation failed for post delete", "error", err)
		return utility.ErrBadRequest
	}

	post := &entity.Post{}

	if err := s.UserRepository.IsAdmin(tx, auth.ID); err != nil {
		if err := s.PostRepository.FindByIDAndUserID(tx, post, request.ID, auth.ID); err != nil {
			slog.Error("Failed to find post by ID and UserID for delete", "error", err)
			return utility.ErrNotFound
		}
	} else {
		if err := s.PostRepository.FindByID(tx, post, request.ID); err != nil {
			slog.Error("Failed to find post by ID for delete", "error", err)
			return utility.ErrNotFound
		}
	}

	if err := s.PostRepository.Delete(tx, post); err != nil {
		slog.Error("Failed to delete post", "error", err)
		return utility.ErrInternalServer
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error("Failed to commit transaction for post delete", "error", err)
		return utility.ErrInternalServer
	}

	return nil
}
