package service

import (
	"chronoverseapi/internal/adapter"
	"chronoverseapi/internal/entity"
	"chronoverseapi/internal/model"
	"chronoverseapi/internal/repository"
	"chronoverseapi/internal/utility"
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type PostService struct {
	DB                 *gorm.DB
	PostRepository     *repository.PostRepository
	UserRepository     *repository.UserRepository
	FileRepository     *repository.FileRepository
	CategoryRepository *repository.CategoryRepository
	FileStorage        *adapter.FileStorage
	Validator          *validator.Validate
	Config             *viper.Viper
}

func NewPostService(
	db *gorm.DB,
	postRepository *repository.PostRepository,
	userRepository *repository.UserRepository,
	fileRepository *repository.FileRepository,
	categoryRepository *repository.CategoryRepository,
	fileStorage *adapter.FileStorage,
	validator *validator.Validate,
	config *viper.Viper,
) *PostService {
	return &PostService{
		DB:                 db,
		PostRepository:     postRepository,
		UserRepository:     userRepository,
		FileRepository:     fileRepository,
		CategoryRepository: categoryRepository,
		FileStorage:        fileStorage,
		Validator:          validator,
		Config:             config,
	}
}

func (s *PostService) Search(ctx context.Context, request *model.PostSearch) (*[]model.PostResponse, *model.Pagination, error) {
	db := s.DB.WithContext(ctx)

	var posts []entity.Post
	total, err := s.PostRepository.Search(db, request, &posts)
	if err != nil {
		slog.Error(err.Error())
		return nil, nil, utility.ErrInternalServerError
	}

	if len(posts) == 0 {
		return &[]model.PostResponse{}, &model.Pagination{}, nil
	}

	var response []model.PostResponse
	for _, post := range posts {
		response = append(response, model.PostResponse{
			ID:            post.ID,
			Title:         post.Title,
			Summary:       post.Summary,
			PublishedDate: post.PublishedDate,
			LastUpdated:   post.LastUpdated,
			Thumbnail:     post.Thumbnail,
			User: &model.UserResponse{
				ID:   post.User.ID,
				Name: post.User.Name,
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

func (s *PostService) Get(ctx context.Context, request *model.PostGet) (*model.PostResponse, error) {
	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrBadRequest
	}

	db := s.DB.WithContext(ctx)

	post := &entity.Post{}
	if err := s.PostRepository.FindByID(db, post, request.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrNotFound
	}

	response := &model.PostResponse{
		ID:            post.ID,
		Title:         post.Title,
		Summary:       post.Summary,
		Content:       post.Content,
		PublishedDate: post.PublishedDate,
		Thumbnail:     post.Thumbnail,
		User: &model.UserResponse{
			ID:             post.User.ID,
			Name:           post.User.Name,
			ProfilePicture: post.User.ProfilePicture,
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

	if err := s.UserRepository.FindById(tx, &entity.User{}, request.UserID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrNotFound
	}

	if err := s.CategoryRepository.FindById(tx, &entity.Category{}, request.CategoryID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrNotFound
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(request.Content))
	if err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	var fileDatas []model.FileData
	var fileNames []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	var once sync.Once
	errChan := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// parallel untuk decoding dan validasi file
	doc.Find("img").Each(func(i int, g *goquery.Selection) {
		src, exists := g.Attr("src")
		if exists && strings.HasPrefix(src, "data:image/") {
			wg.Add(1)
			go func(src string, g *goquery.Selection) {
				defer wg.Done()

				select {
				case <-ctx.Done():
					return
				default:
					file, name, err := utility.Base64ToFile(src)
					if err != nil {
						slog.Error(err.Error())
						once.Do(func() {
							errChan <- utility.ErrBadRequest
							cancel()
						})
						return
					}
					mu.Lock()
					fileDatas = append(fileDatas, model.FileData{File: file, Name: name})
					g.SetAttr("src", name)
					mu.Unlock()
				}
			}(src, g)
		}
	})

	wg.Wait()

	select {
	case err := <-errChan:
		return nil, err
	default:
	}

	// parallel untuk kompresi file dan write file ke temp dir
	if len(fileDatas) > 0 {
		for _, file := range fileDatas {
			wg.Add(1)
			go func(file model.FileData) {
				defer wg.Done()

				select {
				case <-ctx.Done():
					return
				default:
					name, err := utility.CompressImage(file, os.TempDir())
					if err != nil {
						slog.Error(err.Error())
						once.Do(func() {
							errChan <- utility.ErrInternalServerError
							cancel()
						})
						return
					}

					mu.Lock()
					fileNames = append(fileNames, name)
					mu.Unlock()
				}
			}(file)
		}

		wg.Wait()

		select {
		case err := <-errChan:
			return nil, err
		default:
		}
	}

	// defer agar memastikan tempfile dihapus sebelum return
	defer func() {
		var wg sync.WaitGroup
		for _, fileName := range fileNames {
			wg.Add(1)
			go func(fileName string) {
				defer wg.Done()
				if err := s.FileStorage.Delete(filepath.Join(os.TempDir(), fileName)); err != nil {
					slog.Error(err.Error())
				}
			}(fileName)
		}
		wg.Wait()
	}()

	newContent, err := doc.Html()
	if err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}
	request.Content = newContent

	post := &entity.Post{
		Title:         request.Title,
		Summary:       request.Summary,
		Content:       request.Content,
		PublishedDate: time.Now().Unix(),
		UserID:        request.UserID,
		CategoryID:    request.CategoryID,
	}

	if request.Thumbnail != nil {
		post.Thumbnail = utility.CreateFileName(request.Thumbnail)
	}

	if err := s.PostRepository.Create(tx, post); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
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
			return nil, utility.ErrInternalServerError
		}
	}

	if request.Thumbnail != nil {
		if err := s.FileStorage.Store(request.Thumbnail, s.Config.GetString("storage.post")+post.Thumbnail); err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrInternalServerError
		}
	}
	fmt.Println(fileNames)
	// parallel untuk copy file dari temp dir ke dir utama
	if len(fileNames) > 0 {
		for _, fileName := range fileNames {
			wg.Add(1)
			go func(fileName string) {
				defer wg.Done()

				select {
				case <-ctx.Done():
					return
				default:
					if err := s.FileStorage.Copy(fileName, os.TempDir(), s.Config.GetString("storage.post")); err != nil {
						slog.Error(err.Error())
						once.Do(func() {
							errChan <- utility.ErrInternalServerError
							cancel()
						})
					}
				}
			}(fileName)
		}

		wg.Wait()

		select {
		case err := <-errChan:
			return nil, err
		default:
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
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
		if err := s.PostRepository.FindByIDAndUserID(tx, post, request.UserID, auth.ID); err != nil {
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

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(request.Content))
	if err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	var newFileDatas []model.FileData
	var newFileNames []string
	var oldFileNames []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	var once sync.Once
	errChan := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// memproses gambar baru dalam konten
	doc.Find("img").Each(func(i int, g *goquery.Selection) {
		src, exists := g.Attr("src")
		if exists {
			if strings.HasPrefix(src, "data:image/") {
				wg.Add(1)
				go func(src string, g *goquery.Selection) {
					defer wg.Done()

					select {
					case <-ctx.Done():
						return
					default:
						file, name, err := utility.Base64ToFile(src)
						if err != nil {
							slog.Error(err.Error())
							once.Do(func() {
								errChan <- utility.ErrBadRequest
								cancel()
							})
							return
						}
						mu.Lock()
						newFileDatas = append(newFileDatas, model.FileData{File: file, Name: name})
						g.SetAttr("src", name)
						mu.Unlock()
					}
				}(src, g)
			} else {
				oldFileNames = append(oldFileNames, src)
			}
		}
	})

	wg.Wait()

	select {
	case err := <-errChan:
		return nil, err
	default:
	}

	// kompresi gambar baru
	if len(newFileDatas) > 0 {
		for _, file := range newFileDatas {
			wg.Add(1)
			go func(file model.FileData) {
				defer wg.Done()

				select {
				case <-ctx.Done():
					return
				default:
					name, err := utility.CompressImage(file, os.TempDir())
					if err != nil {
						slog.Error(err.Error())
						once.Do(func() {
							errChan <- utility.ErrInternalServerError
							cancel()
						})
						return
					}
					mu.Lock()
					newFileNames = append(newFileNames, name)
					mu.Unlock()
				}
			}(file)
		}

		wg.Wait()

		select {
		case err := <-errChan:
			return nil, err
		default:
		}
	}
	// defer agar memastikan tempfile dihapus sebelum return
	defer func() {
		var wg sync.WaitGroup
		for _, newFileName := range newFileNames {
			wg.Add(1)
			go func(fileName string) {
				defer wg.Done()
				if err := s.FileStorage.Delete(filepath.Join(os.TempDir(), fileName)); err != nil {
					slog.Error(err.Error())
				}
			}(newFileName)
		}
		wg.Wait()
	}()

	newContent, err := doc.Html()
	if err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}
	request.Content = newContent

	oldThumbnail := post.Thumbnail

	post.Title = request.Title
	post.Summary = request.Summary
	post.Content = request.Content
	post.LastUpdated = time.Now().Unix()
	post.CategoryID = request.CategoryID

	if request.UserID != 0 {
		post.UserID = request.UserID
	} else {
		post.UserID = auth.ID
	}

	if request.Thumbnail != nil {
		post.Thumbnail = utility.CreateFileName(request.Thumbnail)
	}

	if err := s.PostRepository.Update(tx, post); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	// ambil daftar file yang tidak lagi digunakan
	unusedFiles, err := s.FileRepository.FindUnusedFile(tx, post.ID, append(oldFileNames, newFileNames...))
	if err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}
	fmt.Println(unusedFiles)

	// hapus file yang tidak digunakan
	if len(unusedFiles) > 0 {
		var unusedFileNames []string
		for _, file := range unusedFiles {
			unusedFileNames = append(unusedFileNames, file.Name)
		}

		err = s.FileRepository.DeleteUnusedFile(tx, post.ID, unusedFileNames)
		if err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrInternalServerError
		}

		// hapus file dari sistem penyimpanan
		for _, file := range unusedFiles {
			wg.Add(1)
			go func(file entity.File) {
				defer wg.Done()

				select {
				case <-ctx.Done():
					return
				default:
					if err := s.FileStorage.Delete(s.Config.GetString("storage.post") + file.Name); err != nil {
						slog.Error(err.Error())
						once.Do(func() {
							errChan <- utility.ErrInternalServerError
							cancel()
						})
					}
				}
			}(file)
		}

		wg.Wait()

		select {
		case err := <-errChan:
			return nil, err
		default:
		}
	}

	// simpan file baru ke database
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
			return nil, utility.ErrInternalServerError
		}
	}

	if request.Thumbnail != nil && oldThumbnail != "" {
		if err := s.FileStorage.Delete(s.Config.GetString("storage.post") + oldThumbnail); err != nil {
			slog.Error(err.Error())
		}
	}

	// simpan thumbnail baru
	if request.Thumbnail != nil {
		if err := s.FileStorage.Store(request.Thumbnail, s.Config.GetString("storage.post")+post.Thumbnail); err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrInternalServerError
		}
	}

	// pindahkan file baru dari temp ke direktori utama
	if len(newFileNames) > 0 {
		for _, fileName := range newFileNames {
			wg.Add(1)
			go func(fileName string) {
				defer wg.Done()

				select {
				case <-ctx.Done():
					return
				default:
					if err := s.FileStorage.Copy(fileName, os.TempDir(), s.Config.GetString("storage.post")); err != nil {
						slog.Error(err.Error())
						once.Do(func() {
							errChan <- utility.ErrInternalServerError
							cancel()
						})
					}
				}
			}(fileName)
		}

		wg.Wait()

		select {
		case err := <-errChan:
			return nil, err
		default:
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
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
		if err := s.PostRepository.FindByIDAndUserID(tx, post, auth.ID, request.ID); err != nil {
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
		return utility.ErrInternalServerError
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
					if err := s.FileStorage.Delete(s.Config.GetString("storage.post") + fileName); err != nil {
						slog.Error(err.Error())
						once.Do(func() {
							errChan <- utility.ErrInternalServerError
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
		if err := s.FileStorage.Delete(s.Config.GetString("storage.post") + post.Thumbnail); err != nil {
			slog.Error(err.Error())
			return utility.ErrInternalServerError
		}
	}

	// hapus semua entri file dari database
	if len(fileNames) > 0 {
		if err := s.FileRepository.DeleteUnusedFile(tx, post.ID, fileNames); err != nil {
			slog.Error(err.Error())
			return utility.ErrInternalServerError
		}
	}

	if err := s.PostRepository.Delete(tx, post); err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	return nil
}
