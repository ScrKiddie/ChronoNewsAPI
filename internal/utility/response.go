package utility

import (
	"encoding/json"
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
	w.WriteHeader(statusCode)
	errorResponse := ResponseError{Error: message}
	json.NewEncoder(w).Encode(errorResponse)
}

func CreateSuccessResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.WriteHeader(statusCode)
	successResponse := ResponseSuccess{Data: data}
	json.NewEncoder(w).Encode(successResponse)
}

func CreateSuccessResponseWithPagination(w http.ResponseWriter, statusCode int, data interface{}, pagination interface{}) {
	w.WriteHeader(statusCode)
	paginationResponse := PaginationResponse{
		Data:       data,
		Pagination: pagination,
	}
	json.NewEncoder(w).Encode(paginationResponse)
}
