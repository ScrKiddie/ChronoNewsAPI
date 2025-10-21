package controller

import (
	"chrononewsapi/internal/model"
	"chrononewsapi/internal/service"
	"chrononewsapi/internal/utility"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type CategoryController struct {
	CategoryService *service.CategoryService
}

func NewCategoryController(categoryService *service.CategoryService) *CategoryController {
	return &CategoryController{CategoryService: categoryService}
}

// Create handles the creation of a new category
// @Summary Create a new category
// @Description Create a new category with the provided details
// @Tags Category
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param category body model.CategoryCreate true "Category data"
// @Success 201 {object} utility.ResponseSuccess{data=model.CategoryResponse}
// @Failure 400 {object} utility.ResponseError
// @Failure 403 {object} utility.ResponseError
// @Failure 409 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/category [post]
func (c *CategoryController) Create(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)

	request := new(model.CategoryCreate)

	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		slog.Error("Failed to decode create category request", "error", err)
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	response, err := c.CategoryService.Create(r.Context(), request, auth)
	if err != nil {
		utility.HandleError(w, err)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusCreated, response)
}

// Get handles retrieving a specific category by ID
// @Summary Get category by ID
// @Description Retrieve details of a category by its ID
// @Tags Category
// @Produce json
// @Param id path int true "Category ID"
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} utility.ResponseSuccess{data=model.CategoryResponse}
// @Failure 400 {object} utility.ResponseError
// @Failure 403 {object} utility.ResponseError
// @Failure 404 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/category/{id} [get]
func (c *CategoryController) Get(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)

	id, err := utility.ToInt32(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error("Failed to parse category ID from URL", "error", err)
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	request := new(model.CategoryGet)
	request.ID = id

	response, err := c.CategoryService.Get(r.Context(), request, auth)
	if err != nil {
		utility.HandleError(w, err)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, response)
}

// Update handles updating a category's details
// @Summary Update a category
// @Description Update the details of a specific category
// @Tags Category
// @Accept json
// @Produce json
// @Param id path int true "Category ID"
// @Param Authorization header string true "Bearer token"
// @Param category body model.CategoryUpdate true "Category data"
// @Success 200 {object} utility.ResponseSuccess{data=model.CategoryResponse}
// @Failure 400 {object} utility.ResponseError
// @Failure 403 {object} utility.ResponseError
// @Failure 404 {object} utility.ResponseError
// @Failure 409 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/category/{id} [put]
func (c *CategoryController) Update(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)

	id, err := utility.ToInt32(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error("Failed to parse category ID from URL", "error", err)
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	request := new(model.CategoryUpdate)
	request.ID = id

	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		slog.Error("Failed to decode update category request", "error", err)
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	response, err := c.CategoryService.Update(r.Context(), request, auth)
	if err != nil {
		utility.HandleError(w, err)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, response)
}

// Delete handles deleting a category
// @Summary Delete a category
// @Description Delete a specific category by ID
// @Tags Category
// @Produce json
// @Param id path int true "Category ID"
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} utility.ResponseSuccess
// @Failure 400 {object} utility.ResponseError
// @Failure 403 {object} utility.ResponseError
// @Failure 404 {object} utility.ResponseError
// @Failure 409 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/category/{id} [delete]
func (c *CategoryController) Delete(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)
	id, err := utility.ToInt32(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error("Failed to parse category ID from URL", "error", err)
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	request := new(model.CategoryDelete)
	request.ID = id

	if err := c.CategoryService.Delete(r.Context(), request, auth); err != nil {
		utility.HandleError(w, err)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, "Category deleted successfully")
}

// List handles retrieving a list of categories
// @Summary List all categories
// @Description Retrieve a list of all categories
// @Tags Category
// @Produce json
// @Success 200 {object} utility.ResponseSuccess{data=[]model.CategoryResponse}
// @Failure 500 {object} utility.ResponseError
// @Router /api/category [get]
func (c *CategoryController) List(w http.ResponseWriter, r *http.Request) {
	response, err := c.CategoryService.List(r.Context())
	if err != nil {
		utility.HandleError(w, err)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, response)
}
