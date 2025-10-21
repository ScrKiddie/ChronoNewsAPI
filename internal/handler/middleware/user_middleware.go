package middleware

import (
	"chrononewsapi/internal/model"
	"chrononewsapi/internal/service"
	"chrononewsapi/internal/utility"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

type UserMiddleware struct {
	UserService *service.UserService
}

func NewUserMiddleware(userService *service.UserService) *UserMiddleware {
	return &UserMiddleware{userService}
}

func (m *UserMiddleware) Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			utility.CreateErrorResponse(w, utility.ErrUnauthorized.Code, utility.ErrUnauthorized.Message)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		auth := &model.Auth{Token: token}

		authResult, err := m.UserService.Verify(r.Context(), auth)
		if err != nil {
			var customErr *utility.CustomError
			if errors.As(err, &customErr) {
				slog.Warn("authorization failed", "error", customErr.Message, "path", r.URL.Path)
				utility.CreateErrorResponse(w, customErr.Code, customErr.Message)
				return
			}
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "auth", authResult)))
	})
}
