package controller

import (
	"chronoverseapi/internal/model"
	"chronoverseapi/internal/service"
	"chronoverseapi/internal/utility"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
)

type CategoryController struct {
	CategoryService *service.CategoryService
}

func NewCategoryController(categoryService *service.CategoryService) *CategoryController {
	return &CategoryController{CategoryService: categoryService}
}

func (c *CategoryController) Create(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)

	request := new(model.CategoryCreate)

	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		slog.Error(err.Error())
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	response, err := c.CategoryService.Create(r.Context(), request, auth)
	if err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusCreated, response)
}

func (c *CategoryController) Get(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)

	id, err := utility.ToInt32(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error(err.Error())
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	request := new(model.CategoryGet)
	request.ID = id

	response, err := c.CategoryService.Get(r.Context(), request, auth)
	if err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, response)
}

func (c *CategoryController) Update(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)

	id, err := utility.ToInt32(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error(err.Error())
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	request := new(model.CategoryUpdate)
	request.ID = id

	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		slog.Error(err.Error())
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	response, err := c.CategoryService.Update(r.Context(), request, auth)
	if err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, response)
}

func (c *CategoryController) Delete(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)
	id, err := utility.ToInt32(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error(err.Error())
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	request := new(model.CategoryDelete)
	request.ID = id

	if err := c.CategoryService.Delete(r.Context(), request, auth); err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, "Category deleted successfully")
}

func (c *CategoryController) List(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)
	response, err := c.CategoryService.List(r.Context(), auth)
	if err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, response)
}
