package controller

import (
	"chrononewsapi/internal/model"
	"chrononewsapi/internal/service"
	"chrononewsapi/internal/utility"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type PostController struct {
	PostService *service.PostService
}

func NewPostController(postService *service.PostService) *PostController {
	return &PostController{PostService: postService}
}

// Search handles searching for posts
// @Summary Search posts
// @Description Search posts with various filtering options
// @Tags Post
// @Produce json
// @Param page query int false "Page number" default(0)
// @Param size query int false "Page size" default(5)
// @Param userID query int false "User ID" default(0)
// @Param title query string false "Title search query"
// @Param userName query string false "User name search query"
// @Param summary query string false "Summary search query"
// @Param categoryName query string false "Category name search query"
// @Param sort query string false "Sort by: view_count, -view_count, created_at, -created_at"
// @Param startDate query int false "Filter posts published after this date (timestamp)"
// @Param endDate query int false "Filter posts published before this date (timestamp)"
// @Param excludeIds query string false "Comma-separated list of post IDs to exclude"
// @Success 200 {object} utility.PaginationResponse{data=[]model.PostResponseWithPreload,pagination=[]model.Pagination}
// @Failure 400 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/post [get]
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
	startDate, err := utility.ToInt64(r.URL.Query().Get("startDate"))
	if err != nil {
		startDate = 0
	}
	endDate, err := utility.ToInt64(r.URL.Query().Get("endDate"))
	if err != nil {
		endDate = 0
	}

	request := &model.PostSearch{
		Page:         page,
		Size:         size,
		Title:        r.URL.Query().Get("title"),
		UserID:       userID,
		UserName:     r.URL.Query().Get("userName"),
		Summary:      r.URL.Query().Get("summary"),
		CategoryName: r.URL.Query().Get("categoryName"),
		Sort:         r.URL.Query().Get("sort"),
		StartDate:    startDate,
		EndDate:      endDate,
		ExcludeIDs:   r.URL.Query().Get("excludeIds"),
	}
	posts, pagination, err := c.PostService.Search(r.Context(), request)
	if err != nil {
		utility.HandleError(w, err)
		return
	}

	utility.CreateSuccessResponseWithPagination(w, http.StatusOK, posts, pagination)
}

// Get handles getting a specific post by ID
// @Summary Get a post by ID
// @Description Retrieve a specific post by its ID
// @Tags Post
// @Produce json
// @Param id path int true "Post ID"
// @Success 200 {object} utility.ResponseSuccess{data=model.PostResponseWithPreload}
// @Failure 400 {object} utility.ResponseError
// @Failure 404 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/post/{id} [get]
func (c *PostController) Get(w http.ResponseWriter, r *http.Request) {
	id, err := utility.ToInt32(chi.URLParam(r, "id"))

	if err != nil {
		slog.Error("Failed to parse post ID from URL", "error", err)
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	request := &model.PostGet{ID: id}

	post, err := c.PostService.Get(r.Context(), request)
	if err != nil {
		utility.HandleError(w, err)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, post)
}

// IncrementViewCount handles incrementing the view count of a post
// @Summary Increment post view count
// @Description Increment the view count for a specific post by its ID
// @Tags Post
// @Produce json
// @Param id path int true "Post ID"
// @Success 200 {object} utility.ResponseSuccess
// @Failure 400 {object} utility.ResponseError
// @Failure 404 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/post/{id}/view [patch]
func (c *PostController) IncrementViewCount(w http.ResponseWriter, r *http.Request) {
	id, err := utility.ToInt32(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error("Failed to parse post ID from URL", "error", err)
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	request := &model.PostIncrementView{ID: id}

	err = c.PostService.IncrementViewCount(r.Context(), request)
	if err != nil {
		utility.HandleError(w, err)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, "View count incremented successfully")
}

// Create handles creating a new post
// @Summary Create a new post
// @Description Create a new post with the given details
// @Tags Post
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param title formData string true "Post Title"
// @Param summary formData string true "Post Summary"
// @Param content formData string true "Post Content"
// @Param userID formData int32 false "User ID"
// @Param categoryID formData int32 true "Category ID"
// @Param thumbnail formData file false "Post Thumbnail"
// @Success 201 {object} utility.ResponseSuccess{data=model.PostResponse}
// @Failure 400 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/post [post]
func (c *PostController) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 350*1024*1024)
	err := r.ParseMultipartForm(350 * 1024 * 1024)
	if err != nil {
		utility.CreateErrorResponse(w, utility.ErrRequestEntityTooLarge.Code, utility.ErrRequestEntityTooLarge.Message)
		return
	}

	auth := r.Context().Value("auth").(*model.Auth)
	categoryID, err := utility.ToInt32(r.FormValue("categoryID"))
	if err != nil {
		slog.Error("Failed to parse categoryID from form", "error", err)
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	var userID int32
	if r.FormValue("userID") != "" {
		userID, err = utility.ToInt32(r.FormValue("userID"))
		if err != nil {
			slog.Error("Failed to parse userID from form", "error", err)
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
		utility.HandleError(w, err)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusCreated, post)
}

// Update handles updating a specific post
// @Summary Update an existing post
// @Description Update an existing post's details
// @Tags Post
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "Post ID"
// @Param title formData string true "Post Title"
// @Param summary formData string true "Post Summary"
// @Param content formData string true "Post Content"
// @Param userID formData int32 false "User ID"
// @Param categoryID formData int32 true "Category ID"
// @Param thumbnail formData file false "Post Thumbnail"
// @Param deleteThumbnail formData bool false "Delete Thumbnail"
// @Success 200 {object} utility.ResponseSuccess{data=model.PostResponse}
// @Failure 400 {object} utility.ResponseError
// @Failure 404 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/post/{id} [put]
func (c *PostController) Update(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 350*1024*1024)
	err := r.ParseMultipartForm(350 * 1024 * 1024)
	if err != nil {
		utility.CreateErrorResponse(w, utility.ErrRequestEntityTooLarge.Code, utility.ErrRequestEntityTooLarge.Message)
		return
	}

	auth := r.Context().Value("auth").(*model.Auth)

	categoryID, err := utility.ToInt32(r.FormValue("categoryID"))
	if err != nil {
		slog.Error("Failed to parse categoryID from form", "error", err)
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	var userID int32

	if r.FormValue("userID") != "" {
		userID, err = utility.ToInt32(r.FormValue("userID"))
		if err != nil {
			slog.Error("Failed to parse userID from form", "error", err)
			utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
			return
		}
	}

	id, err := utility.ToInt32(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error("Failed to parse post ID from URL", "error", err)
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
	request.DeleteThumbnail = r.FormValue("deleteThumbnail") == "true"
	post, err := c.PostService.Update(r.Context(), request, auth)
	if err != nil {
		utility.HandleError(w, err)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, post)
}

// Delete handles deleting a specific post
// @Summary Delete a post
// @Description Delete a post by its ID
// @Tags Post
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "Post ID"
// @Success 200 {object} utility.ResponseSuccess
// @Failure 400 {object} utility.ResponseError
// @Failure 404 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/post/{id} [delete]
func (c *PostController) Delete(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)
	id, err := utility.ToInt32(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error("Failed to parse post ID from URL", "error", err)
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	request := &model.PostDelete{ID: id}

	err = c.PostService.Delete(r.Context(), request, auth)
	if err != nil {
		utility.HandleError(w, err)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, "Post deleted successfully")
}
