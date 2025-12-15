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
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"math"
	"net/http"
	"path/filepath"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserService struct {
	DB              *gorm.DB
	UserRepository  *repository.UserRepository
	PostRepository  *repository.PostRepository
	FileRepository  *repository.FileRepository
	ResetRepository *repository.ResetRepository
	StorageAdapter  *adapter.StorageAdapter
	CaptchaAdapter  *adapter.CaptchaAdapter
	EmailAdapter    *adapter.EmailAdapter
	Validator       *validator.Validate
	Config          *config.Config
}

func NewUserService(db *gorm.DB, userRepository *repository.UserRepository, postRepository *repository.PostRepository, fileRepository *repository.FileRepository, resetRepository *repository.ResetRepository, storageAdapter *adapter.StorageAdapter, captchaAdapter *adapter.CaptchaAdapter, emailAdapter *adapter.EmailAdapter, validator *validator.Validate, config *config.Config) *UserService {
	return &UserService{
		DB:              db,
		UserRepository:  userRepository,
		PostRepository:  postRepository,
		FileRepository:  fileRepository,
		ResetRepository: resetRepository,
		StorageAdapter:  storageAdapter,
		CaptchaAdapter:  captchaAdapter,
		EmailAdapter:    emailAdapter,
		Validator:       validator,
		Config:          config,
	}
}

func (s *UserService) Login(ctx context.Context, request *model.UserLogin) (*model.Auth, error) {
	if err := s.Validator.Struct(request); err != nil {
		slog.Error("Validation failed for user login", "error", err)
		return nil, utility.ErrBadRequest
	}

	captchaRequest := &model.CaptchaRequest{
		TokenCaptcha: request.TokenCaptcha,
		Secret:       s.Config.Captcha.Secret,
	}

	ok, err := s.CaptchaAdapter.Verify(captchaRequest)
	if err != nil {
		slog.Error("Failed to verify captcha", "error", err)
		return nil, utility.ErrInternalServer
	}
	if !ok {
		return nil, utility.ErrBadRequest
	}

	db := s.DB.WithContext(ctx)

	user := new(entity.User)

	if err := s.UserRepository.FindPasswordByEmail(db, user, request.Email); err != nil {
		slog.Error("Failed to find user by email", "error", err)
		return nil, utility.NewCustomError(401, "Incorrect email or password")
	}

	if !utility.VerifyPassword(user.Password, request.Password) {
		return nil, utility.NewCustomError(401, "Incorrect email or password")
	}

	token, err := utility.CreateJWT(s.Config.JWT.Secret, user.Role, s.Config.JWT.Exp, user.ID)
	if err != nil {
		slog.Error("Failed to create JWT", "error", err)
		return nil, utility.ErrInternalServer
	}

	return &model.Auth{Token: token}, nil
}

func (s *UserService) Verify(ctx context.Context, request *model.Auth) (*model.Auth, error) {
	auth, err := utility.ValidateJWT(s.Config.JWT.Secret, request.Token)
	if err != nil {
		slog.Error("Failed to validate JWT", "error", err)
		return nil, utility.ErrUnauthorized
	}

	user := new(entity.User)
	db := s.DB.WithContext(ctx)
	if err := s.UserRepository.FindByID(db, user, auth.ID); err != nil {
		slog.Error("Failed to find user by ID during verification", "error", err)
		return nil, utility.ErrUnauthorized
	}

	return auth, nil
}

func (s *UserService) Current(ctx context.Context, request *model.Auth) (*model.UserResponse, error) {
	if err := s.Validator.Struct(request); err != nil {
		slog.Error("Validation failed for auth model", "error", err)
		return nil, utility.ErrUnauthorized
	}

	db := s.DB.WithContext(ctx)
	user := new(entity.User)
	if err := s.UserRepository.FindByID(db, user, request.ID); err != nil {
		slog.Error("Failed to find current user by ID", "error", err)
		return nil, utility.ErrInternalServer
	}

	var profilePicture string
	for _, file := range user.Files {
		if file.Type == constant.FileTypeProfile {
			profilePicture = utility.BuildImageURL(s.Config, s.Config.Storage.Profile, file.Name)
			break
		}
	}

	return &model.UserResponse{
		ID:             user.ID,
		Name:           user.Name,
		ProfilePicture: profilePicture,
		PhoneNumber:    user.PhoneNumber,
		Email:          user.Email,
		Role:           user.Role,
	}, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, request *model.UserUpdateProfile, auth *model.Auth) (*model.UserResponse, error) {
	if err := s.Validator.Struct(request); err != nil {
		slog.Error("Validation failed for user profile update", "error", err)
		return nil, utility.ErrBadRequest
	}

	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if unique := s.UserRepository.FindIDByEmail(tx, request.Email); unique != 0 && unique != auth.ID {
		return nil, utility.NewCustomError(http.StatusConflict, "Email already exists")
	}

	if unique := s.UserRepository.FindIDByPhoneNumber(tx, request.PhoneNumber); unique != 0 && unique != auth.ID {
		return nil, utility.NewCustomError(http.StatusConflict, "Phone number already exist")
	}

	user := new(entity.User)
	if err := s.UserRepository.FindByID(tx, user, auth.ID); err != nil {
		slog.Error("Failed to find user by ID for profile update", "error", err)
		return nil, utility.ErrInternalServer
	}

	if request.DeleteProfilePicture {
		if err := s.FileRepository.UnlinkFilesFromUser(tx, user.ID); err != nil {
			slog.Error("Failed to unlink profile picture from user", "error", err)
			return nil, utility.ErrInternalServer
		}
	}

	var newProfilePictureName string
	var newProfilePictureFile *entity.File

	if request.ProfilePicture != nil {
		if err := s.FileRepository.UnlinkFilesFromUser(tx, user.ID); err != nil {
			slog.Error("Failed to unlink old profile picture from user", "error", err)
			return nil, utility.ErrInternalServer
		}

		newProfilePictureName = utility.CreateFileName(request.ProfilePicture)
		newProfilePictureFile = &entity.File{
			Name:         newProfilePictureName,
			Type:         constant.FileTypeProfile,
			UsedByUserID: &user.ID,
		}
		if err := s.FileRepository.Create(tx, newProfilePictureFile); err != nil {
			slog.Error("Failed to create new profile picture file record", "error", err)
			return nil, utility.ErrInternalServer
		}
	}

	user.Name = request.Name
	user.Email = request.Email
	user.PhoneNumber = request.PhoneNumber

	if err := s.UserRepository.Update(tx, user); err != nil {
		slog.Error("Failed to update user profile", "error", err)
		return nil, utility.ErrInternalServer
	}

	if request.ProfilePicture != nil {
		storagePath := s.Config.Storage.Profile
		fullPath := filepath.Join(storagePath, newProfilePictureName)

		if err := s.StorageAdapter.Store(request.ProfilePicture, fullPath); err != nil {
			slog.Error("Failed to store new profile picture file", "error", err)
			return nil, utility.ErrInternalServer
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error("Failed to commit transaction for user profile update", "error", err)
		return nil, utility.ErrInternalServer
	}

	var profilePictureURL string
	if newProfilePictureFile != nil {
		profilePictureURL = utility.BuildImageURL(s.Config, s.Config.Storage.Profile, newProfilePictureFile.Name)
	}

	return &model.UserResponse{
		ID:             user.ID,
		Name:           user.Name,
		ProfilePicture: profilePictureURL,
		PhoneNumber:    user.PhoneNumber,
		Email:          user.Email,
		Role:           user.Role,
	}, nil
}

func (s *UserService) UpdatePassword(ctx context.Context, request *model.UserUpdatePassword, auth *model.Auth) error {
	if err := s.Validator.Struct(request); err != nil {
		slog.Error("Validation failed for user password update", "error", err)
		return utility.ErrBadRequest
	}

	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	user := new(entity.User)
	if err := s.UserRepository.FindByID(tx, user, auth.ID); err != nil {
		slog.Error("Failed to find user by ID for password update", "error", err)
		return utility.ErrInternalServer
	}

	if !utility.VerifyPassword(user.Password, request.OldPassword) {
		return utility.NewCustomError(401, "Incorrect old password")
	}

	hashedNewPassword, err := utility.HashPassword(request.Password)
	if err != nil {
		slog.Error("Failed to hash new password", "error", err)
		return utility.ErrInternalServer
	}

	user.Password = hashedNewPassword

	if err := s.UserRepository.Update(tx, user); err != nil {
		slog.Error("Failed to update user password", "error", err)
		return utility.ErrInternalServer
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error("Failed to commit transaction for user password update", "error", err)
		return utility.ErrInternalServer
	}

	return nil
}

func (s *UserService) Search(ctx context.Context, request *model.UserSearch, auth *model.Auth) (*[]model.UserResponse, *model.Pagination, error) {
	db := s.DB.WithContext(ctx)

	if err := s.UserRepository.IsAdmin(db, auth.ID); err != nil {
		slog.Error("Failed to check admin status for user search", "error", err)
		return nil, nil, utility.ErrForbidden
	}

	var users []entity.User
	total, err := s.UserRepository.Search(db, request, &users, auth.ID)
	if err != nil {
		slog.Error("Failed to search users", "error", err)
		return nil, nil, utility.ErrInternalServer
	}

	if len(users) == 0 {
		return &[]model.UserResponse{}, &model.Pagination{}, nil
	}

	var response []model.UserResponse
	for _, v := range users {
		var profilePicture string
		for _, file := range v.Files {
			if file.Type == constant.FileTypeProfile {
				profilePicture = utility.BuildImageURL(s.Config, s.Config.Storage.Profile, file.Name)
				break
			}
		}
		response = append(response, model.UserResponse{
			ID:             v.ID,
			Name:           v.Name,
			ProfilePicture: profilePicture,
			PhoneNumber:    v.PhoneNumber,
			Email:          v.Email,
			Role:           v.Role,
		})
	}

	pagination := &model.Pagination{
		TotalItem: total,
	}

	if request.Page != 0 && request.Size != 0 {
		pagination.Page = request.Page
		pagination.Size = request.Size
		pagination.TotalPage = int64(math.Ceil(float64(total) / float64(request.Size)))
	} else {
		pagination.TotalPage = 1
	}

	return &response, pagination, nil
}

func (s *UserService) Get(ctx context.Context, request *model.UserGet, auth *model.Auth) (*model.UserResponse, error) {
	db := s.DB.WithContext(ctx)

	if err := s.UserRepository.IsAdmin(db, auth.ID); err != nil {
		slog.Error("Failed to check admin status for user get", "error", err)
		return nil, utility.ErrForbidden
	}

	if request.ID == auth.ID {
		return nil, utility.ErrNotFound
	}

	if err := s.Validator.Struct(request); err != nil {
		slog.Error("Validation failed for user get", "error", err)
		return nil, utility.ErrBadRequest
	}

	user := new(entity.User)
	if err := s.UserRepository.FindByID(db, user, request.ID); err != nil {
		slog.Error("Failed to find user by ID", "error", err)
		return nil, utility.ErrNotFound
	}

	var profilePicture string
	for _, file := range user.Files {
		if file.Type == constant.FileTypeProfile {
			profilePicture = utility.BuildImageURL(s.Config, s.Config.Storage.Profile, file.Name)
			break
		}
	}

	return &model.UserResponse{
		ID:             user.ID,
		Name:           user.Name,
		ProfilePicture: profilePicture,
		PhoneNumber:    user.PhoneNumber,
		Email:          user.Email,
		Role:           user.Role,
	}, nil
}

//go:embed template/registered_user_email.html
var registeredUserTemplate embed.FS

func (s *UserService) Create(ctx context.Context, request *model.UserCreate, auth *model.Auth) (*model.UserResponse, error) {
	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.UserRepository.IsAdmin(tx, auth.ID); err != nil {
		slog.Error("Failed to check admin status for user create", "error", err)
		return nil, utility.ErrForbidden
	}

	if err := s.Validator.Struct(request); err != nil {
		slog.Error("Validation failed for user create", "error", err)
		return nil, utility.ErrBadRequest
	}

	if unique := s.UserRepository.FindIDByEmail(tx, request.Email); unique != 0 {
		return nil, utility.NewCustomError(http.StatusConflict, "Email already exists")
	}

	if unique := s.UserRepository.FindIDByPhoneNumber(tx, request.PhoneNumber); unique != 0 {
		return nil, utility.NewCustomError(http.StatusConflict, "Phone number already exist")
	}

	user := &entity.User{
		Name:        request.Name,
		Email:       request.Email,
		PhoneNumber: request.PhoneNumber,
		Role:        request.Role,
	}

	if err := s.UserRepository.Create(tx, user); err != nil {
		slog.Error("Failed to create user", "error", err)
		return nil, utility.ErrInternalServer
	}

	var profilePictureName string
	if request.ProfilePicture != nil {
		profilePictureName = utility.CreateFileName(request.ProfilePicture)
		profilePictureFile := &entity.File{
			Name:         profilePictureName,
			Type:         constant.FileTypeProfile,
			UsedByUserID: &user.ID,
		}
		if err := s.FileRepository.Create(tx, profilePictureFile); err != nil {
			slog.Error("Failed to create profile picture file record", "error", err)
			return nil, utility.ErrInternalServer
		}
	}

	code := uuid.New().String()
	expiredAt := time.Now().Add(time.Hour * time.Duration(s.Config.Reset.Exp)).Unix()
	reset := &entity.Reset{UserID: user.ID}
	err := s.ResetRepository.FindByUserID(tx, reset, user.ID)
	reset.Code = code
	reset.ExpiredAt = expiredAt

	if err := s.ResetRepository.Create(tx, reset); err != nil {
		slog.Error("Failed to create reset token for new user", "error", err)
		return nil, utility.ErrInternalServer
	}

	resetURL := fmt.Sprintf("%s%s?code=%s", s.Config.Web.ClientURL, s.Config.Web.ClientPaths.Reset, code)
	forgotURL := fmt.Sprintf("%s%s", s.Config.Web.ClientURL, s.Config.Web.ClientPaths.Forgot)

	emailBody := &model.EmailBodyData{
		Code:            code,
		ResetURL:        template.URL(resetURL),
		ResetRequestURL: template.URL(forgotURL),
		Year:            time.Now().Year(),
		Expired:         s.Config.Reset.Exp,
	}

	bodyContent, err := utility.GenerateEmailBody(registeredUserTemplate, "template/registered_user_email.html", emailBody)
	if err != nil {
		slog.Error("Failed to generate registered user email body", "error", err)
		return nil, utility.ErrInternalServer
	}

	emailRequest := &model.EmailData{
		To:        request.Email,
		Body:      bodyContent,
		SMTPHost:  s.Config.SMTP.Host,
		SMTPPort:  s.Config.SMTP.Port,
		FromName:  s.Config.SMTP.From.Name,
		FromEmail: s.Config.SMTP.From.Email,
		Username:  s.Config.SMTP.Username,
		Password:  s.Config.SMTP.Password,
		Subject:   "Pendaftaran Akun Berhasil - " + s.Config.SMTP.From.Name,
	}

	if err := s.EmailAdapter.Send(emailRequest); err != nil {
		slog.Error("Failed to send registered user email", "error", err)
		return nil, utility.ErrInternalServer
	}

	if request.ProfilePicture != nil {
		destinationPath := filepath.Join(s.Config.Storage.Profile, profilePictureName)
		if err := s.StorageAdapter.Store(request.ProfilePicture, destinationPath); err != nil {
			slog.Error("Failed to store profile picture for new user", "error", err)
			return nil, utility.ErrInternalServer
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error("Failed to commit transaction for user create", "error", err)
		return nil, utility.ErrInternalServer
	}

	return &model.UserResponse{
		ID:             user.ID,
		Name:           user.Name,
		ProfilePicture: utility.BuildImageURL(s.Config, s.Config.Storage.Profile, profilePictureName),
		PhoneNumber:    user.PhoneNumber,
		Email:          user.Email,
		Role:           user.Role,
	}, nil
}

func (s *UserService) Update(ctx context.Context, request *model.UserUpdate, auth *model.Auth) (*model.UserResponse, error) {
	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.UserRepository.IsAdmin(tx, auth.ID); err != nil {
		slog.Error("Failed to check admin status for user update", "error", err)
		return nil, utility.ErrForbidden
	}

	if request.ID == auth.ID {
		return nil, utility.ErrNotFound
	}

	if err := s.Validator.Struct(request); err != nil {
		slog.Error("Validation failed for user update", "error", err)
		return nil, utility.ErrBadRequest
	}

	user := new(entity.User)
	if err := s.UserRepository.FindByID(tx, user, request.ID); err != nil {
		slog.Error("Failed to find user by ID for update", "error", err)
		return nil, utility.ErrNotFound
	}

	if request.Email != user.Email {
		if unique := s.UserRepository.FindIDByEmail(tx, request.Email); unique != 0 {
			return nil, utility.NewCustomError(http.StatusConflict, "Email already exists")
		}
	}

	if request.PhoneNumber != user.PhoneNumber {
		if unique := s.UserRepository.FindIDByPhoneNumber(tx, request.PhoneNumber); unique != 0 {
			return nil, utility.NewCustomError(http.StatusConflict, "Phone number already exists")
		}
	}

	if request.DeleteProfilePicture {
		if err := s.FileRepository.UnlinkFilesFromUser(tx, user.ID); err != nil {
			slog.Error("Failed to unlink profile picture from user", "error", err)
			return nil, utility.ErrInternalServer
		}
	}

	var newProfilePictureName string
	var newProfilePictureFile *entity.File

	if request.ProfilePicture != nil {
		if err := s.FileRepository.UnlinkFilesFromUser(tx, user.ID); err != nil {
			slog.Error("Failed to unlink old profile picture from user", "error", err)
			return nil, utility.ErrInternalServer
		}

		newProfilePictureName = utility.CreateFileName(request.ProfilePicture)
		newProfilePictureFile = &entity.File{
			Name:         newProfilePictureName,
			Type:         constant.FileTypeProfile,
			UsedByUserID: &user.ID,
		}
		if err := s.FileRepository.Create(tx, newProfilePictureFile); err != nil {
			slog.Error("Failed to create new profile picture file record", "error", err)
			return nil, utility.ErrInternalServer
		}
	}

	user.Name = request.Name
	user.Email = request.Email
	user.PhoneNumber = request.PhoneNumber
	user.Role = request.Role

	if request.Password != "" {
		hashedPassword, err := utility.HashPassword(request.Password)
		if err != nil {
			slog.Error("Failed to hash new password on user update", "error", err)
			return nil, utility.ErrInternalServer
		}
		user.Password = hashedPassword
	}

	if err := s.UserRepository.Update(tx, user); err != nil {
		slog.Error("Failed to update user", "error", err)
		return nil, utility.ErrInternalServer
	}

	if request.ProfilePicture != nil {
		destinationPath := filepath.Join(s.Config.Storage.Profile, newProfilePictureName)
		if err := s.StorageAdapter.Store(
			request.ProfilePicture,
			destinationPath,
		); err != nil {
			slog.Error("Failed to store new profile picture on user update", "error", err)
			return nil, utility.ErrInternalServer
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error("Failed to commit transaction for user update", "error", err)
		return nil, utility.ErrInternalServer
	}

	var profilePictureURL string
	if newProfilePictureFile != nil {
		profilePictureURL = utility.BuildImageURL(s.Config, s.Config.Storage.Profile, newProfilePictureFile.Name)
	}

	return &model.UserResponse{
		ID:             user.ID,
		Name:           user.Name,
		ProfilePicture: profilePictureURL,
		PhoneNumber:    user.PhoneNumber,
		Email:          user.Email,
		Role:           user.Role,
	}, nil
}

func (s *UserService) Delete(ctx context.Context, request *model.UserDelete, auth *model.Auth) error {
	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.UserRepository.IsAdmin(tx, auth.ID); err != nil {
		slog.Error("Failed to check admin status for user delete", "error", err)
		return utility.ErrForbidden
	}

	if request.ID == auth.ID {
		return utility.ErrNotFound
	}

	if err := s.Validator.Struct(request); err != nil {
		slog.Error("Validation failed for user delete", "error", err)
		return utility.ErrBadRequest
	}

	ok, err := s.PostRepository.ExistsByUserID(tx, request.ID)
	if err != nil {
		slog.Error("Failed to check if user is used by post", "error", err)
		return utility.ErrInternalServer
	} else if ok {
		return utility.NewCustomError(http.StatusConflict, "User is used in a post")
	}

	user := new(entity.User)
	if err := s.UserRepository.FindByID(tx, user, request.ID); err != nil {
		slog.Error("Failed to find user by ID for delete", "error", err)
		return utility.ErrNotFound
	}

	if err := s.FileRepository.UnlinkFilesFromUser(tx, user.ID); err != nil {
		slog.Error("Failed to unlink files from user", "error", err)
		return utility.ErrInternalServer
	}

	if err := s.UserRepository.Delete(tx, user); err != nil {
		slog.Error("Failed to delete user", "error", err)
		return utility.ErrInternalServer
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error("Failed to commit transaction for user delete", "error", err)
		return utility.ErrInternalServer
	}

	return nil
}
