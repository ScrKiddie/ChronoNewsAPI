package service

import (
	"chrononewsapi/internal/adapter"
	"chrononewsapi/internal/config"
	"chrononewsapi/internal/entity"
	"chrononewsapi/internal/model"
	"chrononewsapi/internal/repository"
	"chrononewsapi/internal/utility"
	"context"
	"embed"
	"html/template"
	"log/slog"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ResetService struct {
	DB              *gorm.DB
	ResetRepository *repository.ResetRepository
	UserRepository  *repository.UserRepository
	EmailAdapter    *adapter.EmailAdapter
	CaptchaAdapter  *adapter.CaptchaAdapter
	Validator       *validator.Validate
	Config          *config.Config
}

func NewResetService(
	db *gorm.DB,
	resetRepository *repository.ResetRepository,
	userRepository *repository.UserRepository,
	emailAdapter *adapter.EmailAdapter,
	captchaAdapter *adapter.CaptchaAdapter,
	validator *validator.Validate,
	config *config.Config,
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
		slog.Error("Validation failed for reset email request", "error", err)
		return utility.ErrBadRequest
	}

	captchaRequest := &model.CaptchaRequest{
		TokenCaptcha: request.TokenCaptcha,
		Secret:       s.Config.Captcha.Secret,
	}

	ok, err := s.CaptchaAdapter.Verify(captchaRequest)
	if err != nil {
		slog.Error("Failed to verify captcha for reset email", "error", err)
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
	expiredAt := time.Now().Add(time.Hour * time.Duration(s.Config.Reset.Exp)).Unix()

	reset := &entity.Reset{UserID: id}
	err = s.ResetRepository.FindByUserID(tx, reset, id)
	reset.Code = code
	reset.ExpiredAt = expiredAt

	if err != nil {
		slog.Error("Failed to find reset token by user ID, attempting to create new one", "error", err)
		if err := s.ResetRepository.Create(tx, reset); err != nil {
			slog.Error("Failed to create reset token", "error", err)
			return utility.ErrInternalServer
		}
	} else {
		if err := s.ResetRepository.Update(tx, reset); err != nil {
			slog.Error("Failed to update reset token", "error", err)
			return utility.ErrInternalServer
		}
	}

	resetURL := s.Config.Reset.URL + "?" + s.Config.Reset.Query + "=" + code

	emailBody := &model.EmailBodyData{
		Code:            code,
		ResetURL:        template.URL(resetURL),
		ResetRequestURL: template.URL(s.Config.Reset.RequestURL),
		Year:            time.Now().Year(),
		Expired:         s.Config.Reset.Exp,
	}

	bodyContent, err := utility.GenerateEmailBody(resetPasswordTemplate, "template/reset_password_email.html", emailBody)
	if err != nil {
		slog.Error("Failed to generate reset password email body", "error", err)
		return utility.ErrInternalServer
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
		Subject:   "Permintaan Reset Password - " + s.Config.SMTP.From.Name,
	}

	if err := s.EmailAdapter.Send(emailRequest); err != nil {
		slog.Error("Failed to send reset password email", "error", err)
		return utility.ErrInternalServer
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error("Failed to commit transaction for reset email", "error", err)
		return utility.ErrInternalServer
	}

	return nil
}

func (s *ResetService) Reset(ctx context.Context, request *model.ResetRequest) error {
	if err := s.Validator.Struct(request); err != nil {
		slog.Error("Validation failed for reset password request", "error", err)
		return utility.ErrBadRequest
	}

	tx := s.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	reset := &entity.Reset{}
	if err := s.ResetRepository.FindByCode(tx, reset, request.Code); err != nil {
		slog.Error("Failed to find reset token by code", "error", err)
		return utility.ErrNotFound
	}

	if reset.ExpiredAt < time.Now().Unix() {
		if err := s.ResetRepository.Delete(tx, reset); err != nil {
			slog.Error("Failed to delete expired reset token", "error", err)
			return utility.ErrInternalServer
		}
		if err := tx.Commit().Error; err != nil {
			slog.Error("Failed to commit transaction for deleting expired reset token", "error", err)
			return utility.ErrInternalServer
		}
		return utility.ErrBadRequest
	}

	user := new(entity.User)
	user.ID = reset.UserID

	hashedPassword, err := utility.HashPassword(request.Password)
	if err != nil {
		slog.Error("Failed to hash new password", "error", err)
		return utility.ErrInternalServer
	}

	user.Password = hashedPassword

	if err := s.UserRepository.Updates(tx, user); err != nil {
		slog.Error("Failed to update user password after reset", "error", err)
		return utility.ErrInternalServer
	}

	if err := s.ResetRepository.Delete(tx, reset); err != nil {
		slog.Error("Failed to delete used reset token", "error", err)
		return utility.ErrInternalServer
	}

	if err := tx.Commit().Error; err != nil {
		slog.Error("Failed to commit transaction for password reset", "error", err)
		return utility.ErrInternalServer
	}

	return nil
}
