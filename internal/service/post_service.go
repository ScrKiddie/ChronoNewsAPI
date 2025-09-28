package service

import (
	"chrononewsapi/internal/adapter"
	"chrononewsapi/internal/entity"
	"chrononewsapi/internal/model"
	"chrononewsapi/internal/repository"
	"chrononewsapi/internal/utility"
	"context"
	"log/slog"
	"math"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
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
	Config             *viper.Viper
}

func NewPostService(
	db *gorm.DB,
	postRepository *repository.PostRepository,
	userRepository *repository.UserRepository,
	fileRepository *repository.FileRepository,
	categoryRepository *repository.CategoryRepository,
	storageAdapter *adapter.StorageAdapter,
	validator *validator.Validate,
	config *viper.Viper,
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

	var posts []entity.Post
	total, err := s.PostRepository.Search(db, request, &posts)
	if err != nil {
		slog.Error(err.Error())
		return nil, nil, utility.ErrInternalServer
	}

	if len(posts) == 0 {
		return &[]model.PostResponseWithPreload{}, &model.Pagination{}, nil
	}

	var response []model.PostResponseWithPreload
	for _, post := range posts {
		response = append(response, model.PostResponseWithPreload{
			ID:        post.ID,
			Title:     post.Title,
			Summary:   post.Summary,
			CreatedAt: post.CreatedAt,
			UpdatedAt: post.UpdatedAt,
			Thumbnail: post.Thumbnail,
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
		Page:      request.Page,
		Size:      request.Size,
		TotalItem: total,
		TotalPage: int64(math.Ceil(float64(total) / float64(request.Size))),
	}

	return &response, &pagination, nil
}

func (s *PostService) Get(ctx context.Context, request *model.PostGet) (*model.PostResponseWithPreload, error) {
	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrBadRequest
	}

	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	post := &entity.Post{}
	if err := s.PostRepository.FindByID(tx, post, request.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrNotFound
	}

	post.ViewCount = post.ViewCount + 1
	if err := s.PostRepository.Update(tx, post); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrNotFound
	}

	fileIDs, err := utility.ExtractFileIDsFromContent(post.Content)
	if err != nil {
		slog.Error("Failed to extract file IDs from content", "error", err)
		return nil, utility.ErrInternalServer
	}
	fileMap := s.FileRepository.FindAsMap(tx, fileIDs)

	rebuiltContent, err := utility.RebuildContentWithImageSrc(post.Content, fileMap)
	if err != nil {
		slog.Error("Failed to rebuild content with image src", "error", err)
		return nil, utility.ErrInternalServer
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServer
	}

	response := &model.PostResponseWithPreload{
		ID:        post.ID,
		Title:     post.Title,
		Summary:   post.Summary,
		CreatedAt: post.CreatedAt,
		UpdatedAt: post.UpdatedAt,
		Content:   rebuiltContent,
		ViewCount: post.ViewCount,
		Thumbnail: post.Thumbnail,
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
	}

	return response, nil
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
		slog.Error(err.Error())
		return nil, utility.ErrBadRequest
	}

	if err := s.UserRepository.FindByID(tx, &entity.User{}, request.UserID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrNotFound
	}

	if err := s.CategoryRepository.FindById(tx, &entity.Category{}, request.CategoryID); err != nil {
		slog.Error(err.Error())
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

	if request.Thumbnail != nil {
		post.Thumbnail = utility.CreateFileName(request.Thumbnail)
	}

	if err := s.PostRepository.Create(tx, post); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServer
	}

	if err := s.FileRepository.LinkFilesToPost(tx, fileIDs, post.ID); err != nil {
		slog.Error("Failed to link files to post", "error", err)
		return nil, utility.ErrInternalServer
	}

	if request.Thumbnail != nil {
		if err := s.StorageAdapter.Store(request.Thumbnail, s.Config.GetString("storage.post")+post.Thumbnail); err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrInternalServer
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
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
		Thumbnail:  post.Thumbnail,
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
			slog.Error(err.Error())
			return nil, utility.ErrNotFound
		}
	} else {
		if err := s.PostRepository.FindByID(tx, post, request.ID); err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrNotFound
		}
	}

	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrBadRequest
	}

	if err := s.CategoryRepository.FindById(tx, &entity.Category{}, request.CategoryID); err != nil {
		slog.Error(err.Error())
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

	oldThumbnail := post.Thumbnail
	post.Title = request.Title
	post.Summary = request.Summary
	post.Content = sanitizedContent
	post.CategoryID = request.CategoryID

	if request.UserID != 0 {
		post.UserID = request.UserID
	} else {
		post.UserID = auth.ID
	}

	if request.DeleteThumbnail {
		post.Thumbnail = ""
	}

	if request.Thumbnail != nil {
		post.Thumbnail = utility.CreateFileName(request.Thumbnail)
	}

	if err := s.PostRepository.Update(tx, post); err != nil {
		slog.Error(err.Error())
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

	if (request.Thumbnail != nil || request.DeleteThumbnail) && oldThumbnail != "" {
		if err := s.StorageAdapter.Delete(s.Config.GetString("storage.post") + oldThumbnail); err != nil {
			slog.Error(err.Error())
		}
	}

	if request.Thumbnail != nil {
		if err := s.StorageAdapter.Store(request.Thumbnail, s.Config.GetString("storage.post")+post.Thumbnail); err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrInternalServer
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
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
		Thumbnail:  post.Thumbnail,
	}

	return response, nil
}

func (s *PostService) Delete(ctx context.Context, request *model.PostDelete, auth *model.Auth) error {
	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return utility.ErrBadRequest
	}

	post := &entity.Post{}

	if err := s.UserRepository.IsAdmin(tx, auth.ID); err != nil {
		if err := s.PostRepository.FindByIDAndUserID(tx, post, request.ID, auth.ID); err != nil {
			slog.Error(err.Error())
			return utility.ErrNotFound
		}
	} else {
		if err := s.PostRepository.FindByID(tx, post, request.ID); err != nil {
			slog.Error(err.Error())
			return utility.ErrNotFound
		}
	}

	if post.Thumbnail != "" {
		if err := s.StorageAdapter.Delete(s.Config.GetString("storage.post") + post.Thumbnail); err != nil {
			slog.Error("Failed to delete post thumbnail from storage", "error", err)
		}
	}

	if err := s.PostRepository.Delete(tx, post); err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServer
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServer
	}

	return nil
}
