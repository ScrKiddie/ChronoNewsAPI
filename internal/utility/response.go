package utility

import (
	"encoding/json"
	"net/http"
)

func CreateErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
	})
}
func CreateSuccessResponse(w http.ResponseWriter, statusCode int, data any) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": data,
	})
}
func CreateSuccessResponseWithPagination(w http.ResponseWriter, statusCode int, data any, pagination any) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":       data,
		"pagination": pagination,
	})
}
