package service

import (
	"chronoverseapi/internal/adapter"
	"chronoverseapi/internal/entity"
	"chronoverseapi/internal/model"
	"chronoverseapi/internal/repository"
	"chronoverseapi/internal/utility"
	"context"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"html/template"
	"log/slog"
	"time"
)

type ResetService struct {
	DB              *gorm.DB
	ResetRepository *repository.ResetRepository
	UserRepository  *repository.UserRepository
	Email           *adapter.EmailAdapter
	Validator       *validator.Validate
	Config          *viper.Viper
}

func NewResetService(
	db *gorm.DB,
	resetRepository *repository.ResetRepository,
	userRepository *repository.UserRepository,
	email *adapter.EmailAdapter,
	validator *validator.Validate,
	config *viper.Viper,
) *ResetService {
	return &ResetService{
		DB:              db,
		ResetRepository: resetRepository,
		UserRepository:  userRepository,
		Validator:       validator,
		Config:          config,
	}
}

func (s *ResetService) ResetEmail(ctx context.Context, request *model.ResetEmailRequest) error {
	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return nil
	}

	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	id := s.UserRepository.FindIDByEmail(tx, request.Email)
	if id == 0 {
		return nil
	}

	code := uuid.New().String()
	expiredAt := time.Now().Add(time.Hour * time.Duration(s.Config.GetInt("reset.exp"))).Unix()

	reset := &entity.Reset{
		UserID:    id,
		Code:      code,
		ExpiredAt: expiredAt,
	}

	if err := s.ResetRepository.Create(tx, reset); err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServer
	}

	resetURL := s.Config.GetString("reset.url") + "?" + s.Config.GetString("reset.query") + "=" + code

	emailBody := &model.EmailBodyData{
		Code:            code,
		ResetURL:        template.URL(resetURL),
		ResetRequestURL: template.URL(s.Config.GetString("reset.request.url")),
		Year:            time.Now().Year(),
		Expired:         s.Config.GetInt("reset.exp"),
	}

	bodyContent, err := utility.GenerateEmailBody("./internal/template/reset_password_email.html", emailBody)
	if err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServer
	}

	emailRequest := &model.EmailData{
		To:        request.Email,
		Body:      bodyContent,
		SMTPHost:  s.Config.GetString("smtp.host"),
		SMTPPort:  s.Config.GetInt("smtp.port"),
		FromName:  s.Config.GetString("smtp.from.name"),
		FromEmail: s.Config.GetString("smtp.from.email"),
		Username:  s.Config.GetString("smtp.username"),
		Password:  s.Config.GetString("smtp.password"),
		Subject:   "Permintaan Reset Password - " + s.Config.GetString("smtp.from.name"),
	}

	if err := s.Email.Send(emailRequest); err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServer
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServer
	}

	return nil
}
