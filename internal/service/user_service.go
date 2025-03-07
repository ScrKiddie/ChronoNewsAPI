package service

import (
	"chronoverseapi/internal/adapter"
	"chronoverseapi/internal/constant"
	"chronoverseapi/internal/entity"
	"chronoverseapi/internal/model"
	"chronoverseapi/internal/repository"
	"chronoverseapi/internal/utility"
	"context"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"log/slog"
	"math"
	"net/http"
)

type UserService struct {
	DB             *gorm.DB
	UserRepository *repository.UserRepository
	PostRepository *repository.PostRepository
	FileStorage    *adapter.FileStorage
	Captcha        *adapter.Captcha
	Validator      *validator.Validate
	Config         *viper.Viper
}

func NewUserService(db *gorm.DB, userRepository *repository.UserRepository, postRepository *repository.PostRepository, fileStorage *adapter.FileStorage, captcha *adapter.Captcha, validator *validator.Validate, config *viper.Viper) *UserService {
	return &UserService{
		DB:             db,
		UserRepository: userRepository,
		PostRepository: postRepository,
		FileStorage:    fileStorage,
		Captcha:        captcha,
		Validator:      validator,
		Config:         config,
	}
}

func (s *UserService) Register(ctx context.Context, request *model.UserRegister) error {
	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return utility.ErrBadRequest
	}

	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if unique := s.UserRepository.FindIDByEmail(tx, request.Email); unique != 0 {
		return utility.NewCustomError(http.StatusConflict, "Email already exists")
	}

	if unique := s.UserRepository.FindIDByPhoneNumber(tx, request.PhoneNumber); unique != 0 {
		return utility.NewCustomError(http.StatusConflict, "Phone number already exist")
	}

	hashedPassword, err := utility.HashPassword(request.Password)
	if err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	user := &entity.User{
		Name:        request.Name,
		PhoneNumber: request.PhoneNumber,
		Email:       request.Email,
		Password:    hashedPassword,
		Role:        constant.User,
	}

	if err := s.UserRepository.Create(tx, user); err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	return nil
}

func (s *UserService) Login(ctx context.Context, request *model.UserLogin) (*model.Auth, error) {
	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.NewCustomError(401, "Email atau password salah")
	}

	captchaRequest := &model.CaptchaRequest{
		TokenCaptcha: request.TokenCaptcha,
		Secret:       s.Config.GetString("captcha.secret"),
	}

	ok, err := s.Captcha.Verify(captchaRequest)
	if err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}
	if !ok {
		return nil, utility.ErrBadRequest
	}

	db := s.DB.WithContext(ctx)

	user := new(entity.User)

	if err := s.UserRepository.FindPasswordByEmail(db, user, request.Email); err != nil {
		slog.Error(err.Error())
		return nil, utility.NewCustomError(401, "Email atau password salah")
	}

	if !utility.VerifyPassword(user.Password, request.Password) {
		return nil, utility.NewCustomError(401, "Email atau password salah")
	}

	token, err := utility.CreateJWT(s.Config.GetString("jwt.secret"), user.Role, s.Config.GetInt("jwt.exp"), user.ID)
	if err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	return &model.Auth{Token: token}, nil
}

func (s *UserService) Verify(ctx context.Context, request *model.Auth) (*model.Auth, error) {
	auth, err := utility.ValidateJWT(s.Config.GetString("jwt.secret"), request.Token)
	if err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrUnauthorized
	}

	user := new(entity.User)
	db := s.DB.WithContext(ctx)
	if err := s.UserRepository.FindById(db, user, auth.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrUnauthorized
	}

	return auth, nil
}

func (s *UserService) Current(ctx context.Context, request *model.Auth) (*model.UserResponse, error) {
	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrUnauthorized
	}

	db := s.DB.WithContext(ctx)
	user := new(entity.User)
	if err := s.UserRepository.FindById(db, user, request.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	return &model.UserResponse{
		ID:             user.ID,
		Name:           user.Name,
		ProfilePicture: user.ProfilePicture,
		PhoneNumber:    user.PhoneNumber,
		Email:          user.Email,
		Role:           user.Role,
	}, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, request *model.UserUpdateProfile, auth *model.Auth) (*model.UserResponse, error) {
	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
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
	if err := s.UserRepository.FindById(tx, user, auth.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	oldFileName := user.ProfilePicture

	if request.ProfilePicture != nil {
		user.ProfilePicture = utility.CreateFileName(request.ProfilePicture)
	}

	user.Name = request.Name
	user.Email = request.Email
	user.PhoneNumber = request.PhoneNumber

	if err := s.UserRepository.Update(tx, user); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	if request.ProfilePicture != nil && oldFileName != "" {
		if err := s.FileStorage.Delete(s.Config.GetString("storage.profile") + oldFileName); err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrInternalServerError
		}
	}

	if request.ProfilePicture != nil {
		if err := s.FileStorage.Store(request.ProfilePicture, s.Config.GetString("storage.profile")+user.ProfilePicture); err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrInternalServerError
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	return &model.UserResponse{
		ID:             user.ID,
		Name:           user.Name,
		ProfilePicture: user.ProfilePicture,
		PhoneNumber:    user.PhoneNumber,
		Email:          user.Email,
		Role:           user.Role,
	}, nil
}

func (s *UserService) UpdatePassword(ctx context.Context, request *model.UserUpdatePassword, auth *model.Auth) error {
	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return utility.ErrBadRequest
	}

	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	user := new(entity.User)
	if err := s.UserRepository.FindById(tx, user, auth.ID); err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	if !utility.VerifyPassword(user.Password, request.OldPassword) {
		return utility.NewCustomError(401, "Password lama salah")
	}

	hashedNewPassword, err := utility.HashPassword(request.Password)
	if err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	user.Password = hashedNewPassword

	if err := s.UserRepository.Update(tx, user); err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	return nil
}

func (s *UserService) Search(ctx context.Context, request *model.UserSearch, auth *model.Auth) (*[]model.UserResponse, *model.Pagination, error) {
	db := s.DB.WithContext(ctx)

	if err := s.UserRepository.IsAdmin(db, auth.ID); err != nil {
		slog.Error(err.Error())
		return nil, nil, utility.ErrForbidden
	}

	var users []entity.User
	total, err := s.UserRepository.Search(db, request, &users, auth.ID)
	if err != nil {
		slog.Error(err.Error())
		return nil, nil, utility.ErrInternalServerError
	}

	if len(users) == 0 {
		return &[]model.UserResponse{}, &model.Pagination{}, nil
	}

	var response []model.UserResponse
	for _, v := range users {
		response = append(response, model.UserResponse{
			ID:             v.ID,
			Name:           v.Name,
			ProfilePicture: v.ProfilePicture,
			PhoneNumber:    v.PhoneNumber,
			Email:          v.Email,
			Role:           v.Role,
		})
	}

	var pagination *model.Pagination
	if request.Page != 0 && request.Size != 0 {
		pagination = &model.Pagination{
			Page:      request.Page,
			Size:      request.Size,
			TotalItem: total,
			TotalPage: int64(math.Ceil(float64(total) / float64(request.Size))),
		}
	} else {
		pagination = nil
	}

	return &response, pagination, nil
}

func (s *UserService) Get(ctx context.Context, request *model.UserGet, auth *model.Auth) (*model.UserResponse, error) {
	db := s.DB.WithContext(ctx)

	if err := s.UserRepository.IsAdmin(db, auth.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrForbidden
	}

	if request.ID == auth.ID {
		return nil, utility.ErrNotFound
	}

	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrBadRequest
	}

	user := new(entity.User)
	if err := s.UserRepository.FindById(db, user, request.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrNotFound
	}

	return &model.UserResponse{
		ID:             user.ID,
		Name:           user.Name,
		ProfilePicture: user.ProfilePicture,
		PhoneNumber:    user.PhoneNumber,
		Email:          user.Email,
		Role:           user.Role,
	}, nil
}

func (s *UserService) Create(ctx context.Context, request *model.UserCreate, auth *model.Auth) (*model.UserResponse, error) {
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

	if unique := s.UserRepository.FindIDByEmail(tx, request.Email); unique != 0 {
		return nil, utility.NewCustomError(http.StatusConflict, "Email already exists")
	}

	if unique := s.UserRepository.FindIDByPhoneNumber(tx, request.PhoneNumber); unique != 0 {
		return nil, utility.NewCustomError(http.StatusConflict, "Phone number already exist")
	}

	user := new(entity.User)
	if request.ProfilePicture != nil {
		user.ProfilePicture = utility.CreateFileName(request.ProfilePicture)
	}

	hashedPassword, err := utility.HashPassword(request.Password)
	if err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	user.Name = request.Name
	user.Email = request.Email
	user.PhoneNumber = request.PhoneNumber
	user.Password = hashedPassword
	user.Role = request.Role

	if err := s.UserRepository.Update(tx, user); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	if request.ProfilePicture != nil {
		if err := s.FileStorage.Store(request.ProfilePicture, s.Config.GetString("storage.profile")+user.ProfilePicture); err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrInternalServerError
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	return &model.UserResponse{
		ID:             user.ID,
		Name:           user.Name,
		ProfilePicture: user.ProfilePicture,
		PhoneNumber:    user.PhoneNumber,
		Email:          user.Email,
		Role:           user.Role,
	}, nil
}

func (s *UserService) Update(ctx context.Context, request *model.UserUpdate, auth *model.Auth) (*model.UserResponse, error) {
	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.UserRepository.IsAdmin(tx, auth.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrForbidden
	}

	if request.ID == auth.ID {
		return nil, utility.ErrNotFound
	}

	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrBadRequest
	}

	user := new(entity.User)
	if err := s.UserRepository.FindById(tx, user, request.ID); err != nil {
		slog.Error(err.Error())
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

	user.Name = request.Name
	user.Email = request.Email
	user.PhoneNumber = request.PhoneNumber
	user.Role = request.Role
	oldFileName := user.ProfilePicture

	if request.ProfilePicture != nil {
		user.ProfilePicture = utility.CreateFileName(request.ProfilePicture)
	}

	if request.Password != "" {
		hashedPassword, err := utility.HashPassword(request.Password)
		if err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrInternalServerError
		}
		user.Password = hashedPassword
	}

	if err := s.UserRepository.Update(tx, user); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	if request.ProfilePicture != nil && oldFileName != "" {
		if err := s.FileStorage.Delete(s.Config.GetString("storage.profile") + oldFileName); err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrInternalServerError
		}
	}

	if request.ProfilePicture != nil {
		if err := s.FileStorage.Store(
			request.ProfilePicture,
			s.Config.GetString("storage.profile")+user.ProfilePicture,
		); err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrInternalServerError
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	return &model.UserResponse{
		ID:             user.ID,
		Name:           user.Name,
		ProfilePicture: user.ProfilePicture,
		PhoneNumber:    user.PhoneNumber,
		Email:          user.Email,
		Role:           user.Role,
	}, nil
}

func (s *UserService) Delete(ctx context.Context, request *model.UserDelete, auth *model.Auth) error {
	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := s.UserRepository.IsAdmin(tx, auth.ID); err != nil {
		slog.Error(err.Error())
		return utility.ErrForbidden
	}

	if request.ID == auth.ID {
		return utility.ErrNotFound
	}

	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return utility.ErrBadRequest
	}

	ok, err := s.PostRepository.ExistsByUserID(tx, request.ID)
	if err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	} else if ok {
		return utility.NewCustomError(http.StatusConflict, "User digunakan pada berita")
	}

	user := new(entity.User)
	if err := s.UserRepository.FindById(tx, user, request.ID); err != nil {
		slog.Error(err.Error())
		return utility.ErrNotFound
	}

	if err := s.UserRepository.Delete(tx, user); err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	if user.ProfilePicture != "" {
		if err := s.FileStorage.Delete(s.Config.GetString("storage.profile") + user.ProfilePicture); err != nil {
			slog.Error(err.Error())
			return utility.ErrInternalServerError
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	return nil
}
