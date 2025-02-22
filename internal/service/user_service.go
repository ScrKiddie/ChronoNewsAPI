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
	FileStorage    *adapter.FileStorage
	Validator      *validator.Validate
	Config         *viper.Viper
}

func NewUserService(db *gorm.DB, userRepository *repository.UserRepository, fileStorage *adapter.FileStorage, validator *validator.Validate, config *viper.Viper) *UserService {
	return &UserService{
		DB:             db,
		UserRepository: userRepository,
		FileStorage:    fileStorage,
		Validator:      validator,
		Config:         config,
	}
}

func (u *UserService) Register(ctx context.Context, request *model.UserRegister) error {
	if err := u.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return utility.ErrBadRequest
	}

	tx := u.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if unique := u.UserRepository.FindIDByEmail(tx, request.Email); unique != 0 {
		return utility.NewCustomError(http.StatusConflict, "Email already exists")
	}

	if unique := u.UserRepository.FindIDByPhoneNumber(tx, request.PhoneNumber); unique != 0 {
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

	if err := u.UserRepository.Create(tx, user); err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	return nil
}

func (u *UserService) Login(ctx context.Context, request *model.UserLogin) (*model.Auth, error) {
	if err := u.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.NewCustomError(401, "Email atau password salah")
	}

	db := u.DB.WithContext(ctx)

	user := new(entity.User)

	if request.Email != "" {
		if err := u.UserRepository.FindPasswordByEmail(db, user, request.Email); err != nil {
			slog.Error(err.Error())
			return nil, utility.NewCustomError(401, "Email atau password salah")
		}
	} else {
		if err := u.UserRepository.FindPasswordByPhoneNumber(db, user, request.PhoneNumber); err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrUnauthorized
		}
	}

	if !utility.VerifyPassword(user.Password, request.Password) {
		return nil, utility.NewCustomError(401, "Email atau password salah")
	}

	token, err := utility.CreateJWT(u.Config.GetString("jwt.secret"), user.Role, u.Config.GetInt("jwt.exp"), user.ID)
	if err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	return &model.Auth{Token: token}, nil
}

func (u *UserService) Verify(ctx context.Context, request *model.Auth) (*model.Auth, error) {
	auth, err := utility.ValidateJWT(u.Config.GetString("jwt.secret"), request.Token)
	if err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrUnauthorized
	}

	user := new(entity.User)
	db := u.DB.WithContext(ctx)
	if err := u.UserRepository.FindById(db, user, auth.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrUnauthorized
	}

	return auth, nil
}

func (u *UserService) Current(ctx context.Context, request *model.Auth) (*model.UserResponse, error) {
	if err := u.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrUnauthorized
	}

	db := u.DB.WithContext(ctx)
	user := new(entity.User)
	if err := u.UserRepository.FindById(db, user, request.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	return &model.UserResponse{
		Name:           user.Name,
		ProfilePicture: user.ProfilePicture,
		PhoneNumber:    user.PhoneNumber,
		Email:          user.Email,
	}, nil
}

func (u *UserService) UpdateProfile(ctx context.Context, request *model.UserUpdateProfile, auth *model.Auth) (*model.UserResponse, error) {
	if err := u.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrBadRequest
	}

	tx := u.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if unique := u.UserRepository.FindIDByEmail(tx, request.Email); unique != 0 && unique != auth.ID {
		return nil, utility.NewCustomError(http.StatusConflict, "Email already exists")
	}

	if unique := u.UserRepository.FindIDByPhoneNumber(tx, request.PhoneNumber); unique != 0 && unique != auth.ID {
		return nil, utility.NewCustomError(http.StatusConflict, "Phone number already exist")
	}

	user := new(entity.User)
	if err := u.UserRepository.FindById(tx, user, auth.ID); err != nil {
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

	if err := u.UserRepository.Update(tx, user); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	if request.ProfilePicture != nil && oldFileName != "" {
		if err := u.FileStorage.Delete(u.Config.GetString("storage.profile") + oldFileName); err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrInternalServerError
		}
	}

	if request.ProfilePicture != nil {
		if err := u.FileStorage.Store(request.ProfilePicture, u.Config.GetString("storage.profile")+user.ProfilePicture); err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrInternalServerError
		}
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	return &model.UserResponse{
		Name:           user.Name,
		ProfilePicture: user.ProfilePicture,
		PhoneNumber:    user.PhoneNumber,
		Email:          user.Email,
	}, nil
}

func (u *UserService) UpdatePassword(ctx context.Context, request *model.UserUpdatePassword, auth *model.Auth) error {
	if err := u.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return utility.ErrBadRequest
	}

	tx := u.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	user := new(entity.User)
	if err := u.UserRepository.FindById(tx, user, auth.ID); err != nil {
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

	if err := u.UserRepository.Update(tx, user); err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	return nil
}

func (u *UserService) Search(ctx context.Context, request *model.UserSearch, auth *model.Auth) (*[]model.UserResponse, *model.Pagination, error) {
	db := u.DB.WithContext(ctx)

	if err := u.UserRepository.IsAdmin(db, auth.ID); err != nil {
		slog.Error(err.Error())
		return nil, nil, utility.ErrForbidden
	}

	var users []entity.User
	total, err := u.UserRepository.SearchNonAdmin(db, request, &users)
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

func (u *UserService) Get(ctx context.Context, request *model.UserGet, auth *model.Auth) (*model.UserResponse, error) {
	db := u.DB.WithContext(ctx)

	if err := u.UserRepository.IsAdmin(db, auth.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrForbidden
	}

	if err := u.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrBadRequest
	}

	user := new(entity.User)
	if err := u.UserRepository.FindNonAdminByID(db, user, request.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrNotFound
	}

	return &model.UserResponse{
		ID:             user.ID,
		Name:           user.Name,
		ProfilePicture: user.ProfilePicture,
		PhoneNumber:    user.PhoneNumber,
		Email:          user.Email,
	}, nil
}

func (u *UserService) Create(ctx context.Context, request *model.UserCreate, auth *model.Auth) (*model.UserResponse, error) {
	tx := u.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := u.UserRepository.IsAdmin(tx, auth.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrForbidden
	}

	if err := u.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrBadRequest
	}

	if unique := u.UserRepository.FindIDByEmail(tx, request.Email); unique != 0 {
		return nil, utility.NewCustomError(http.StatusConflict, "Email already exists")
	}

	if unique := u.UserRepository.FindIDByPhoneNumber(tx, request.PhoneNumber); unique != 0 {
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
	user.Role = constant.Journalist

	if err := u.UserRepository.Update(tx, user); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	if request.ProfilePicture != nil {
		if err := u.FileStorage.Store(request.ProfilePicture, u.Config.GetString("storage.profile")+user.ProfilePicture); err != nil {
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
	}, nil
}

func (u *UserService) Update(ctx context.Context, request *model.UserUpdate, auth *model.Auth) (*model.UserResponse, error) {
	tx := u.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := u.UserRepository.IsAdmin(tx, auth.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrForbidden
	}

	if err := u.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrBadRequest
	}

	user := new(entity.User)
	if err := u.UserRepository.FindNonAdminByID(tx, user, request.ID); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrNotFound
	}

	if request.Email != user.Email {
		if unique := u.UserRepository.FindIDByEmail(tx, request.Email); unique != 0 {
			return nil, utility.NewCustomError(http.StatusConflict, "Email already exists")
		}
	}

	if request.PhoneNumber != user.PhoneNumber {
		if unique := u.UserRepository.FindIDByPhoneNumber(tx, request.PhoneNumber); unique != 0 {
			return nil, utility.NewCustomError(http.StatusConflict, "Phone number already exists")
		}
	}

	user.Name = request.Name
	user.Email = request.Email
	user.PhoneNumber = request.PhoneNumber

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

	if err := u.UserRepository.Update(tx, user); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	if request.ProfilePicture != nil && oldFileName != "" {
		if err := u.FileStorage.Delete(u.Config.GetString("storage.profile") + oldFileName); err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrInternalServerError
		}
	}

	if request.ProfilePicture != nil {
		if err := u.FileStorage.Store(
			request.ProfilePicture,
			u.Config.GetString("storage.profile")+user.ProfilePicture,
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
	}, nil
}

func (u *UserService) Delete(ctx context.Context, request *model.UserDelete, auth *model.Auth) error {
	tx := u.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := u.UserRepository.IsAdmin(tx, auth.ID); err != nil {
		slog.Error(err.Error())
		return utility.ErrForbidden
	}

	if err := u.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return utility.ErrBadRequest
	}

	user := new(entity.User)
	if err := u.UserRepository.FindNonAdminByID(tx, user, request.ID); err != nil {
		slog.Error(err.Error())
		return utility.ErrNotFound
	}

	if err := u.UserRepository.Delete(tx, user); err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	if user.ProfilePicture != "" {
		if err := u.FileStorage.Delete(u.Config.GetString("storage.profile") + user.ProfilePicture); err != nil {
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
