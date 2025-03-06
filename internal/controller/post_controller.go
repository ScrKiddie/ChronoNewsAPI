package controller

import (
	"chronoverseapi/internal/model"
	"chronoverseapi/internal/service"
	"chronoverseapi/internal/utility"
	"fmt"
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
)

type PostController struct {
	PostService *service.PostService
}

func NewPostController(postService *service.PostService) *PostController {
	return &PostController{PostService: postService}
}

func (c *PostController) Search(w http.ResponseWriter, r *http.Request) {
	page, err := utility.ToInt64(r.URL.Query().Get("page"))
	if err != nil {
		page = 0
	}
	size, err := utility.ToInt64(r.URL.Query().Get("size"))
	if err != nil {
		size = 5
	}
	userID, err := utility.ToInt32(r.URL.Query().Get("userID"))
	if err != nil {
		userID = 0
	}

	request := &model.PostSearch{
		Page:         page,
		Size:         size,
		Title:        r.URL.Query().Get("title"),
		UserID:       userID,
		UserName:     r.URL.Query().Get("userName"),
		Summary:      r.URL.Query().Get("summary"),
		CategoryName: r.URL.Query().Get("categoryName"),
	}
	posts, pagination, err := c.PostService.Search(r.Context(), request)
	if err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}

	utility.CreateSuccessResponseWithPagination(w, http.StatusOK, posts, pagination)
}

func (c *PostController) Get(w http.ResponseWriter, r *http.Request) {
	id, err := utility.ToInt32(chi.URLParam(r, "id"))

	if err != nil {
		slog.Error(err.Error())
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	request := &model.PostGet{ID: id}

	post, err := c.PostService.Get(r.Context(), request)
	if err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, post)
}

func (c *PostController) Create(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(1 * 1024 * 1024 * 1024)
	if err != nil {
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, "Request terlalu besar")
		return
	}

	auth := r.Context().Value("auth").(*model.Auth)
	fmt.Println(r.FormValue("categoryID"))
	fmt.Println(r.FormValue("userID"))
	categoryID, err := utility.ToInt32(r.FormValue("categoryID"))
	if err != nil {
		slog.Error(err.Error())
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	var userID int32
	if r.FormValue("userID") != "" {
		userID, err = utility.ToInt32(r.FormValue("userID"))
		if err != nil {
			slog.Error(err.Error())
			utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
			return
		}
	}

	request := &model.PostCreate{
		Title:      r.FormValue("title"),
		Summary:    r.FormValue("summary"),
		Content:    r.FormValue("content"),
		UserID:     userID,
		CategoryID: categoryID,
	}

	_, request.Thumbnail, _ = r.FormFile("thumbnail")

	post, err := c.PostService.Create(r.Context(), request, auth)
	if err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusCreated, post)
}

func (c *PostController) Update(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(1 * 1024 * 1024 * 1024)
	if err != nil {
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, "Request terlalu besar")
		return
	}

	auth := r.Context().Value("auth").(*model.Auth)

	categoryID, err := utility.ToInt32(r.FormValue("categoryID"))
	if err != nil {
		slog.Error(err.Error())
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	var userID int32

	if r.FormValue("userID") != "" {
		userID, err = utility.ToInt32(r.FormValue("userID"))
		if err != nil {
			slog.Error(err.Error())
			utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
			return
		}
	}

	id, err := utility.ToInt32(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error(err.Error())
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	request := &model.PostUpdate{
		ID:         id,
		Title:      r.FormValue("title"),
		Summary:    r.FormValue("summary"),
		Content:    r.FormValue("content"),
		UserID:     userID,
		CategoryID: categoryID,
	}
	_, request.Thumbnail, _ = r.FormFile("thumbnail")

	post, err := c.PostService.Update(r.Context(), request, auth)
	if err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, post)
}

func (c *PostController) Delete(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)
	id, err := utility.ToInt32(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error(err.Error())
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	request := &model.PostDelete{ID: id}

	err = c.PostService.Delete(r.Context(), request, auth)
	if err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, "Post deleted successfully")
}
