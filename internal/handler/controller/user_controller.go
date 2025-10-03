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

type UserController struct {
	UserService *service.UserService
}

func NewUserController(userService *service.UserService) *UserController {
	return &UserController{UserService: userService}
}

// Login handles user login and returns a JWT token
// @Summary User login
// @Description User logs in with email/phone number and password, returning a JWT token
// @Tags User
// @Accept json
// @Produce json
// @Param user body model.UserLogin true "User login data"
// @Success 200 {object} utility.ResponseSuccess
// @Failure 400 {object} utility.ResponseError
// @Failure 401 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/user/login [post]
func (c *UserController) Login(w http.ResponseWriter, r *http.Request) {
	request := new(model.UserLogin)

	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		slog.Error("Failed to decode login request", "error", err)
		utility.CreateErrorResponse(w, utility.ErrUnauthorized.Code, utility.ErrUnauthorized.Message)
		return
	}

	response, err := c.UserService.Login(r.Context(), request)
	if err != nil {
		utility.HandleError(w, err)
		return
	}
	utility.CreateSuccessResponse(w, http.StatusOK, response.Token)
}

// Current retrieves the current logged-in user's profile
// @Summary Get current user's profile
// @Description Get the profile of the currently logged-in user
// @Tags User
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} utility.ResponseSuccess{data=model.UserResponse}
// @Failure 401 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/user/current [get]
func (c *UserController) Current(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)
	response, err := c.UserService.Current(r.Context(), auth)
	if err != nil {
		utility.HandleError(w, err)
		return
	}
	utility.CreateSuccessResponse(w, http.StatusOK, response)
}

// UpdateProfile updates the current user's profile
// @Summary Update current user's profile
// @Description Update the current user's name, phone number, email, and profile picture
// @Tags User
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param name formData string true "User name"
// @Param phoneNumber formData string true "Phone number"
// @Param email formData string true "Email"
// @Param profilePicture formData file false "Profile picture"
// @Param deleteProfilePicture formData bool false "Delete profile picture"
// @Success 200 {object} utility.ResponseSuccess{data=model.UserResponse}
// @Failure 400 {object} utility.ResponseError
// @Failure 401 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/user/current/profile [patch]
func (c *UserController) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)
	request := new(model.UserUpdateProfile)
	request.Name = r.FormValue("name")
	request.PhoneNumber = r.FormValue("phoneNumber")
	request.Email = r.FormValue("email")
	request.DeleteProfilePicture = r.FormValue("deleteProfilePicture") == "true"
	_, request.ProfilePicture, _ = r.FormFile("profilePicture")
	response, err := c.UserService.UpdateProfile(r.Context(), request, auth)
	if err != nil {
		utility.HandleError(w, err)
		return
	}
	utility.CreateSuccessResponse(w, http.StatusOK, response)
}

// UpdatePassword updates the current user's password
// @Summary Update current user's password
// @Description Update the current user's password
// @Tags User
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param password body model.UserUpdatePassword true "New password details"
// @Success 201 {object} utility.ResponseSuccess
// @Failure 400 {object} utility.ResponseError
// @Failure 401 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/user/current/password [patch]
func (c *UserController) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)
	request := new(model.UserUpdatePassword)
	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		slog.Error("Failed to decode update password request", "error", err)
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}
	if err := c.UserService.UpdatePassword(r.Context(), request, auth); err != nil {
		utility.HandleError(w, err)
		return
	}
	utility.CreateSuccessResponse(w, http.StatusCreated, "Password updated successfully")
}

// Search searches for users based on query parameters
// @Summary Search for users
// @Description Search for users by name, role, phone number, and email
// @Tags User
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param name query string false "Name"
// @Param phoneNumber query string false "Phone number"
// @Param email query string false "Email"
// @Param role query string false "Role"
// @Param page query int false "Page number"
// @Param size query int false "Page size"
// @Success 200 {object} utility.PaginationResponse{data=[]model.UserResponse,pagination=[]model.Pagination}
// @Failure 401 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/user [get]
func (c *UserController) Search(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)

	page, err := utility.ToInt64(r.URL.Query().Get("page"))
	if err != nil {
		page = 0
	}
	size, err := utility.ToInt64(r.URL.Query().Get("size"))
	if err != nil {
		size = 0
	}

	request := new(model.UserSearch)
	request.Page = page
	request.Size = size
	request.Name = r.URL.Query().Get("name")
	request.Role = r.URL.Query().Get("role")
	request.PhoneNumber = r.URL.Query().Get("phoneNumber")
	request.Email = r.URL.Query().Get("email")

	response, pagination, err := c.UserService.Search(r.Context(), request, auth)
	if err != nil {
		utility.HandleError(w, err)
		return
	}
	if pagination == nil {
		utility.CreateSuccessResponse(w, http.StatusOK, response)
	} else {
		utility.CreateSuccessResponseWithPagination(w, http.StatusOK, response, pagination)
	}

}

// Get retrieves a specific user by ID
// @Summary Get user by ID
// @Description Retrieve a specific user by their ID
// @Tags User
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "User ID"
// @Success 200 {object} utility.ResponseSuccess{data=model.UserResponse}
// @Failure 400 {object} utility.ResponseError
// @Failure 401 {object} utility.ResponseError
// @Failure 404 {object} utility.ResponseError
// @Router /api/user/{id} [get]
func (c *UserController) Get(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)

	id, err := utility.ToInt32(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error("Failed to parse user ID from URL", "error", err)
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	request := new(model.UserGet)
	request.ID = id

	response, err := c.UserService.Get(r.Context(), request, auth)
	if err != nil {
		utility.HandleError(w, err)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, response)
}

// Create creates a new user
// @Summary Create a new user
// @Description Create a new user with name, phone number, email, and role
// @Tags User
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param name formData string true "User name"
// @Param phoneNumber formData string true "Phone number"
// @Param email formData string true "Email"
// @Param profilePicture formData file false "Profile picture"
// @Param role formData string true "Role (admin, journalist)"
// @Success 201 {object} utility.ResponseSuccess{data=model.UserResponse}
// @Failure 400 {object} utility.ResponseError
// @Failure 409 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/user [post]
func (c *UserController) Create(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)

	request := new(model.UserCreate)
	request.Name = r.FormValue("name")
	request.Role = r.FormValue("role")
	request.PhoneNumber = r.FormValue("phoneNumber")
	request.Email = r.FormValue("email")
	_, request.ProfilePicture, _ = r.FormFile("profilePicture")

	response, err := c.UserService.Create(r.Context(), request, auth)

	if err != nil {
		utility.HandleError(w, err)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusCreated, response)
}

// Update updates an existing user
// @Summary Update user by ID
// @Description Update an existing user's details
// @Tags User
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "User ID"
// @Param name formData string true "User name"
// @Param phoneNumber formData string true "Phone number"
// @Param email formData string true "Email"
// @Param profilePicture formData file false "Profile picture"
// @Param password formData string false "Password"
// @Param role formData string true "Role"
// @Param deleteProfilePicture formData bool false "Delete profile picture"
// @Success 200 {object} utility.ResponseSuccess{data=model.UserResponse}
// @Failure 400 {object} utility.ResponseError
// @Failure 401 {object} utility.ResponseError
// @Failure 404 {object} utility.ResponseError
// @Failure 409 {object} utility.ResponseError
// @Router /api/user/{id} [put]
func (c *UserController) Update(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)

	id, err := utility.ToInt32(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error("Failed to parse user ID from URL", "error", err)
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	request := new(model.UserUpdate)
	request.ID = id
	request.Name = r.FormValue("name")
	request.Role = r.FormValue("role")
	request.PhoneNumber = r.FormValue("phoneNumber")
	request.Email = r.FormValue("email")
	request.Password = r.FormValue("password")
	_, request.ProfilePicture, _ = r.FormFile("profilePicture")
	request.DeleteProfilePicture = r.FormValue("deleteProfilePicture") == "true"
	response, err := c.UserService.Update(r.Context(), request, auth)
	if err != nil {
		utility.HandleError(w, err)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, response)
}

// Delete deletes an existing user
// @Summary Delete user by ID
// @Description Delete an existing user by their ID
// @Tags User
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path int true "User ID"
// @Success 200 {object} utility.ResponseSuccess
// @Failure 400 {object} utility.ResponseError
// @Failure 401 {object} utility.ResponseError
// @Failure 404 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/user/{id} [delete]
func (c *UserController) Delete(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)

	id, err := utility.ToInt32(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error("Failed to parse user ID from URL", "error", err)
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	request := new(model.UserDelete)
	request.ID = id

	if err := c.UserService.Delete(r.Context(), request, auth); err != nil {
		utility.HandleError(w, err)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, "User deleted successfully")
}
