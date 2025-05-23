package service

import (
	"chrononewsapi/internal/adapter"
	"chrononewsapi/internal/entity"
	"chrononewsapi/internal/model"
	"chrononewsapi/internal/repository"
	"chrononewsapi/internal/utility"
	"context"
	"embed"
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
	EmailAdapter    *adapter.EmailAdapter
	CaptchaAdapter  *adapter.CaptchaAdapter
	Validator       *validator.Validate
	Config          *viper.Viper
}

func NewResetService(
	db *gorm.DB,
	resetRepository *repository.ResetRepository,
	userRepository *repository.UserRepository,
	emailAdapter *adapter.EmailAdapter,
	captchaAdapter *adapter.CaptchaAdapter,
	validator *validator.Validate,
	config *viper.Viper,
) *ResetService {
	return &ResetService{
		DB:              db,
		ResetRepository: resetRepository,
		UserRepository:  userRepository,
		Validator:       validator,
		EmailAdapter:    emailAdapter,
		CaptchaAdapter:  captchaAdapter,
		Config:          config,
	}
}

//go:embed template/reset_password_email.html
var resetPasswordTemplate embed.FS

func (s *ResetService) ResetEmail(ctx context.Context, request *model.ResetEmailRequest) error {
	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return utility.ErrBadRequest
	}

	captchaRequest := &model.CaptchaRequest{
		TokenCaptcha: request.TokenCaptcha,
		Secret:       s.Config.GetString("captcha.secret"),
	}

	ok, err := s.CaptchaAdapter.Verify(captchaRequest)
	if err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServer
	}
	if !ok {
		return utility.ErrBadRequest
	}

	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	id := s.UserRepository.FindIDByEmail(tx, request.Email)
	if id == 0 {
		return nil
	}

	code := uuid.New().String()
	expiredAt := time.Now().Add(time.Hour * time.Duration(s.Config.GetInt("reset.exp"))).Unix()

	reset := &entity.Reset{UserID: id}
	err = s.ResetRepository.FindByUserID(tx, reset, id)
	reset.Code = code
	reset.ExpiredAt = expiredAt

	if err != nil {
		slog.Error(err.Error())
		if err := s.ResetRepository.Create(tx, reset); err != nil {
			slog.Error(err.Error())
			return utility.ErrInternalServer
		}
	} else {
		if err := s.ResetRepository.Update(tx, reset); err != nil {
			slog.Error(err.Error())
			return utility.ErrInternalServer
		}
	}

	resetURL := s.Config.GetString("reset.url") + "?" + s.Config.GetString("reset.query") + "=" + code

	emailBody := &model.EmailBodyData{
		Code:            code,
		ResetURL:        template.URL(resetURL),
		ResetRequestURL: template.URL(s.Config.GetString("reset.request.url")),
		Year:            time.Now().Year(),
		Expired:         s.Config.GetInt("reset.exp"),
	}

	bodyContent, err := utility.GenerateEmailBody(resetPasswordTemplate, "template/reset_password_email.html", emailBody)
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

	if err := s.EmailAdapter.Send(emailRequest); err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServer
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServer
	}

	return nil
}

func (s *ResetService) Reset(ctx context.Context, request *model.ResetRequest) error {
	if err := s.Validator.Struct(request); err != nil {
		slog.Error(err.Error())
		return utility.ErrBadRequest
	}

	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	reset := &entity.Reset{}
	if err := s.ResetRepository.FindByCode(tx, reset, request.Code); err != nil {
		slog.Error(err.Error())
		return utility.ErrNotFound
	}

	if reset.ExpiredAt < time.Now().Unix() {
		if err := s.ResetRepository.Delete(tx, reset); err != nil {
			slog.Error(err.Error())
			return utility.ErrInternalServer
		}
		if err := tx.Commit().Error; err != nil {
			slog.Error(err.Error())
			return utility.ErrInternalServer
		}
		return utility.ErrBadRequest
	}

	user := new(entity.User)
	user.ID = reset.UserID

	hashedPassword, err := utility.HashPassword(request.Password)
	if err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServer
	}

	user.Password = hashedPassword

	if err := s.UserRepository.Updates(tx, user); err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServer
	}

	if err := s.ResetRepository.Delete(tx, reset); err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServer
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error(err.Error())
		return utility.ErrInternalServer
	}

	return nil
}
