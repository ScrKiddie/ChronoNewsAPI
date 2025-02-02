package middleware

import (
	"ChronoverseAPI/internal/model"
	"ChronoverseAPI/internal/service"
	"ChronoverseAPI/internal/utility"
	"context"
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
		auth := &model.Authorization{Token: token}

		authResult, err := m.UserService.Verify(r.Context(), auth)
		if err != nil {
			if customErr, ok := err.(*utility.CustomError); ok {
				utility.CreateErrorResponse(w, customErr.Code, customErr.Message)
				return
			}
		}

		slog.Info("authorized request", "id", authResult.ID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "auth", authResult)))
	})
}
