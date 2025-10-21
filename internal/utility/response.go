package utility

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type ResponseSuccess struct {
	Data interface{} `json:"data"`
}

type ResponseError struct {
	Error string `json:"error"`
}

type PaginationResponse struct {
	Data       interface{} `json:"data"`
	Pagination interface{} `json:"pagination"`
}

func CreateErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	errorResponse := ResponseError{Error: message}
	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		slog.Error("Failed to write error response", "err", err)
	}
}

func HandleError(w http.ResponseWriter, err error) {
	var customErr *CustomError
	if errors.As(err, &customErr) {
		CreateErrorResponse(w, customErr.Code, customErr.Message)
	} else {
		slog.Error("An unexpected error occurred", "error", err)
		CreateErrorResponse(w, http.StatusInternalServerError, "An internal server error occurred")
	}
}

func CreateSuccessResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	successResponse := ResponseSuccess{Data: data}
	if err := json.NewEncoder(w).Encode(successResponse); err != nil {
		slog.Error("Failed to write success response", "err", err)
	}
}

func CreateSuccessResponseWithPagination(w http.ResponseWriter, statusCode int, data interface{}, pagination interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	paginationResponse := PaginationResponse{
		Data:       data,
		Pagination: pagination,
	}
	if err := json.NewEncoder(w).Encode(paginationResponse); err != nil {
		slog.Error("Failed to write pagination response", "err", err)
	}
}
