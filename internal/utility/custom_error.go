package utility

import (
	"fmt"
	"net/http"
)

type CustomError struct {
	Code    int
	Message string
}

func NewCustomError(code int, message string) *CustomError {
	return &CustomError{
		Code:    code,
		Message: message,
	}
}

func (e *CustomError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

var (
	ErrInternalServerError = &CustomError{
		Code:    http.StatusInternalServerError,
		Message: "Internal server error",
	}
	ErrBadRequest = &CustomError{
		Code:    http.StatusBadRequest,
		Message: "Bad request",
	}
	ErrUnauthorized = &CustomError{
		Code:    http.StatusUnauthorized,
		Message: "Unauthorized",
	}
	ErrNotFound = &CustomError{
		Code:    http.StatusNotFound,
		Message: "Not found",
	}
	ErrForbidden = &CustomError{
		Code:    http.StatusForbidden,
		Message: "Forbidden",
	}
)
