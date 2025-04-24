package service

import (
	"chrononewsapi/internal/adapter"
	"chrononewsapi/internal/entity"
	"chrononewsapi/internal/model"
	"chrononewsapi/internal/repository"
	"chrononewsapi/internal/utility"
	"context"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
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
			ID:            post.ID,
			Title:         post.Title,
			Summary:       post.Summary,
			PublishedDate: post.PublishedDate,
			LastUpdated:   post.LastUpdated,
			Thumbnail:     post.Thumbnail,
			ViewCount:     post.ViewCount,
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

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServer
	}

	response := &model.PostResponseWithPreload{
		ID:            post.ID,
		Title:         post.Title,
		Summary:       post.Summary,
		Content:       post.Content,
		PublishedDate: post.PublishedDate,
		LastUpdated:   post.LastUpdated,
		Thumbnail:     post.Thumbnail,
		ViewCount:     post.ViewCount,
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

	newContent, fileNames, _, err := utility.HandleParallelContentProcessing(&request.Content)
	if err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServer
	}

	defer func() {
		for _, fileName := range fileNames {
			go func(fileName string) {
				if err := s.StorageAdapter.Delete(filepath.Join(os.TempDir(), fileName)); err != nil {
					slog.Error(err.Error())
				}
			}(fileName)
		}
	}()

	post := &entity.Post{
		Title:         request.Title,
		Summary:       request.Summary,
		Content:       newContent,
		PublishedDate: time.Now().Unix(),
		UserID:        request.UserID,
		CategoryID:    request.CategoryID,
	}

	if request.Thumbnail != nil {
		post.Thumbnail = utility.CreateFileName(request.Thumbnail)
	}

	if err := s.PostRepository.Create(tx, post); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServer
	}

	if len(fileNames) > 0 {
		var fileEntities []entity.File
		for _, fileName := range fileNames {
			fileEntities = append(fileEntities, entity.File{
				PostID: post.ID,
				Name:   fileName,
			})
		}

		if err := tx.Create(&fileEntities).Error; err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrInternalServer
		}
	}

	if request.Thumbnail != nil {
		if err := s.StorageAdapter.Store(request.Thumbnail, s.Config.GetString("storage.post")+post.Thumbnail); err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrInternalServer
		}
	}

	if len(fileNames) > 0 {
		for _, fileName := range fileNames {
			if err := s.StorageAdapter.Copy(fileName, os.TempDir(), s.Config.GetString("storage.post")); err != nil {
				slog.Error(err.Error())
				return nil, utility.ErrInternalServer
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServer
	}

	response := &model.PostResponse{
		ID:            post.ID,
		CategoryID:    post.CategoryID,
		UserID:        post.UserID,
		Title:         post.Title,
		Summary:       post.Summary,
		Content:       post.Content,
		PublishedDate: post.PublishedDate,
		LastUpdated:   post.LastUpdated,
		Thumbnail:     post.Thumbnail,
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

	newContent, newFileNames, oldFileNames, err := utility.HandleParallelContentProcessing(&request.Content)
	if err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServer
	}

	// defer untuk delete temp file
	defer func() {
		for _, fileName := range newFileNames {
			go func(fileName string) {
				if err := s.StorageAdapter.Delete(filepath.Join(os.TempDir(), fileName)); err != nil {
					slog.Error(err.Error())
				}
			}(fileName)
		}
	}()

	oldThumbnail := post.Thumbnail
	post.Title = request.Title
	post.Summary = request.Summary
	post.Content = newContent
	post.LastUpdated = time.Now().Unix()
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

	unusedFiles, err := s.FileRepository.FindUnusedFile(tx, post.ID, append(oldFileNames, newFileNames...))
	if err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServer
	}

	if len(unusedFiles) > 0 {
		var unusedFileNames []string

		for _, file := range unusedFiles {
			unusedFileNames = append(unusedFileNames, file.Name)
		}

		err = s.FileRepository.DeleteUnusedFile(tx, post.ID, unusedFileNames)
		if err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrInternalServer
		}

		for _, file := range unusedFiles {
			if err := s.StorageAdapter.Delete(s.Config.GetString("storage.post") + file.Name); err != nil {
				slog.Error(err.Error())
				return nil, utility.ErrInternalServer
			}
		}
	}

	if len(newFileNames) > 0 {
		var fileEntities []entity.File
		for _, fileName := range newFileNames {
			fileEntities = append(fileEntities, entity.File{
				PostID: post.ID,
				Name:   fileName,
			})
		}
		if err := tx.Create(&fileEntities).Error; err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrInternalServer
		}
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

	if len(newFileNames) > 0 {
		for _, fileName := range newFileNames {
			if err := s.StorageAdapter.Copy(fileName, os.TempDir(), s.Config.GetString("storage.post")); err != nil {
				slog.Error(err.Error())
				return nil, utility.ErrInternalServer
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServer
	}

	response := &model.PostResponse{
		ID:            post.ID,
		CategoryID:    post.CategoryID,
		UserID:        post.UserID,
		Title:         post.Title,
		Summary:       post.Summary,
		Content:       post.Content,
		PublishedDate: post.PublishedDate,
		LastUpdated:   post.LastUpdated,
		Thumbnail:     post.Thumbnail,
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

	var files []entity.File
	if err := s.FileRepository.FindByPostId(tx, &files, post.ID); err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServer
	}

	var fileNames []string
	for _, file := range files {
		fileNames = append(fileNames, file.Name)
	}

	var wg sync.WaitGroup
	var once sync.Once
	errChan := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// hapus file dari penyimpanan secara paralel
	if len(fileNames) > 0 {
		for _, fileName := range fileNames {
			wg.Add(1)
			go func(fileName string) {
				defer wg.Done()

				select {
				case <-ctx.Done():
					return
				default:
					if err := s.StorageAdapter.Delete(s.Config.GetString("storage.post") + fileName); err != nil {
						slog.Error(err.Error())
						once.Do(func() {
							errChan <- utility.ErrInternalServer
							cancel()
						})
					}
				}
			}(fileName)
		}

		wg.Wait()

		select {
		case err := <-errChan:
			return err
		default:
		}
	}

	if post.Thumbnail != "" {
		if err := s.StorageAdapter.Delete(s.Config.GetString("storage.post") + post.Thumbnail); err != nil {
			slog.Error(err.Error())
			return utility.ErrInternalServer
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
