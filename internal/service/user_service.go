package service

import (
	"ChronoverseAPI/internal/adapter"
	"ChronoverseAPI/internal/constant"
	"ChronoverseAPI/internal/entity"
	"ChronoverseAPI/internal/model"
	"ChronoverseAPI/internal/repository"
	"ChronoverseAPI/internal/utility"
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

func (u *UserService) Login(ctx context.Context, request *model.UserLogin) (*model.UserResponse, error) {
	if err := u.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrUnauthorized
	}

	db := u.DB.WithContext(ctx)

	user := new(entity.User)

	if request.Email != "" {
		if err := u.UserRepository.FindPasswordByEmail(db, user, request.Email); err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrUnauthorized
		}
	} else {
		if err := u.UserRepository.FindPasswordByPhoneNumber(db, user, request.PhoneNumber); err != nil {
			slog.Error(err.Error())
			return nil, utility.ErrUnauthorized
		}
	}

	if !utility.VerifyPassword(user.Password, request.Password) {
		return nil, utility.ErrUnauthorized
	}

	token, err := utility.CreateJWT(u.Config.GetString("jwt.secret"), u.Config.GetInt("jwt.exp"), user.ID)
	if err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrInternalServerError
	}

	return &model.UserResponse{Token: token}, nil
}

func (u *UserService) Verify(ctx context.Context, request *model.Authorization) (*model.UserAuthorization, error) {
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

func (u *UserService) Current(ctx context.Context, request *model.UserAuthorization) (*model.UserResponse, error) {
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

func (u *UserService) UpdateProfile(ctx context.Context, request *model.UserUpdateProfile) (*model.UserResponse, error) {
	if err := u.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil, utility.ErrBadRequest
	}

	tx := u.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if unique := u.UserRepository.FindIDByEmail(tx, request.Email); unique != 0 && unique != request.ID {
		return nil, utility.NewCustomError(http.StatusConflict, "Email already exists")
	}

	if unique := u.UserRepository.FindIDByPhoneNumber(tx, request.PhoneNumber); unique != 0 && unique != request.ID {
		return nil, utility.NewCustomError(http.StatusConflict, "Phone number already exist")
	}

	user := new(entity.User)
	if err := u.UserRepository.FindById(tx, user, request.ID); err != nil {
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

func (u *UserService) UpdatePassword(ctx context.Context, request *model.UserUpdatePassword) error {
	if err := u.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return utility.ErrBadRequest
	}

	tx := u.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	user := new(entity.User)
	if err := u.UserRepository.FindById(tx, user, request.ID); err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServerError
	}

	if !utility.VerifyPassword(user.Password, request.OldPassword) {
		return utility.ErrUnauthorized
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

func (u *UserService) Search(ctx context.Context, request *model.UserSearch) (*[]model.UserResponse, *model.Pagination, error) {
	db := u.DB.WithContext(ctx)

	if err := u.UserRepository.IsAdmin(db, request.ID); err != nil {
		slog.Error(err.Error())
		return nil, nil, utility.ErrForbidden
	}

	var users []entity.User
	total, err := u.UserRepository.Search(db, request, &users)
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

//update
//delete
