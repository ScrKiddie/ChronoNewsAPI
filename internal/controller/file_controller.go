package controller

import (
	"chrononewsapi/internal/service"
	"chrononewsapi/internal/utility"
	"log/slog"
	"net/http"
)

type FileController struct {
	FileService *service.FileService
}

func NewFileController(fileService *service.FileService) *FileController {
	return &FileController{FileService: fileService}
}

// UploadImage handles image uploads from the editor
// @Summary Upload an image
// @Description Upload an image to be used in post content
// @Tags File
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param image formData file true "Image file to upload"
// @Success 201 {object} utility.ResponseSuccess{data=model.ImageUploadResponse}
// @Failure 400 {object} utility.ResponseError
// @Failure 413 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/image [post]
func (c *FileController) UploadImage(w http.ResponseWriter, r *http.Request) {
	_, fileHeader, err := r.FormFile("image")
	if err != nil {
		slog.Error("Failed to get file from form", "error", err)
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, "Image file is required")
		return
	}

	response, err := c.FileService.UploadImage(r.Context(), fileHeader)
	if err != nil {
		customErr := err.(*utility.CustomError)
		utility.CreateErrorResponse(w, customErr.Code, customErr.Message)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusCreated, response)
}
