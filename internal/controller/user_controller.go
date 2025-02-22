package controller

import (
	"chronoverseapi/internal/model"
	"chronoverseapi/internal/service"
	"chronoverseapi/internal/utility"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
	"strconv"
)

type UserController struct {
	UserService *service.UserService
}

func NewUserController(userService *service.UserService) *UserController {
	return &UserController{UserService: userService}
}

func (u *UserController) Register(w http.ResponseWriter, r *http.Request) {
	request := new(model.UserRegister)

	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		slog.Error(err.Error())
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	if err := u.UserService.Register(r.Context(), request); err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}
	utility.CreateSuccessResponse(w, http.StatusCreated, "User registered successfully")
}

func (u *UserController) Login(w http.ResponseWriter, r *http.Request) {
	request := new(model.UserLogin)

	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		slog.Error(err.Error())
		utility.CreateErrorResponse(w, utility.ErrUnauthorized.Code, utility.ErrUnauthorized.Message)
		return
	}

	response, err := u.UserService.Login(r.Context(), request)
	if err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}
	utility.CreateSuccessResponse(w, http.StatusOK, response.Token)
}

func (u *UserController) Current(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)
	response, err := u.UserService.Current(r.Context(), auth)
	if err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}
	utility.CreateSuccessResponse(w, http.StatusOK, response)
}

func (u *UserController) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)
	request := new(model.UserUpdateProfile)
	request.Name = r.FormValue("name")
	request.PhoneNumber = r.FormValue("phoneNumber")
	request.Email = r.FormValue("email")
	_, request.ProfilePicture, _ = r.FormFile("profilePicture")
	response, err := u.UserService.UpdateProfile(r.Context(), request, auth)
	if err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}
	utility.CreateSuccessResponse(w, http.StatusOK, response)
}

func (u *UserController) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)
	request := new(model.UserUpdatePassword)
	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		slog.Error(err.Error())
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}
	if err := u.UserService.UpdatePassword(r.Context(), request, auth); err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}
	utility.CreateSuccessResponse(w, http.StatusCreated, "Password updated successfully")
}

func (u *UserController) Search(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		page = 0
	}
	size, err := strconv.Atoi(r.URL.Query().Get("size"))
	if err != nil {
		size = 5
	}

	request := new(model.UserSearch)
	request.Page = int64(page)
	request.Size = int64(size)
	request.Name = r.URL.Query().Get("name")
	request.PhoneNumber = r.URL.Query().Get("phoneNumber")
	request.Email = r.URL.Query().Get("email")

	response, pagination, err := u.UserService.Search(r.Context(), request, auth)
	if err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}
	utility.CreateSuccessResponseWithPagination(w, http.StatusOK, response, pagination)
}

func (u *UserController) Get(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)

	id, err := utility.ToInt32(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error(err.Error())
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	request := new(model.UserGet)
	request.ID = id

	response, err := u.UserService.Get(r.Context(), request, auth)
	if err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, response)
}

func (u *UserController) Create(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)

	request := new(model.UserCreate)
	request.Name = r.FormValue("name")
	request.PhoneNumber = r.FormValue("phoneNumber")
	request.Email = r.FormValue("email")
	request.Password = r.FormValue("password")
	_, request.ProfilePicture, _ = r.FormFile("profilePicture")

	response, err := u.UserService.Create(r.Context(), request, auth)

	if err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusCreated, response)
}

func (u *UserController) Update(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)

	id, err := utility.ToInt32(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error(err.Error())
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	request := new(model.UserUpdate)
	request.ID = id
	request.Name = r.FormValue("name")
	request.PhoneNumber = r.FormValue("phoneNumber")
	request.Email = r.FormValue("email")
	request.Password = r.FormValue("password")
	_, request.ProfilePicture, _ = r.FormFile("profilePicture")

	response, err := u.UserService.Update(r.Context(), request, auth)
	if err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, response)
}

func (u *UserController) Delete(w http.ResponseWriter, r *http.Request) {
	auth := r.Context().Value("auth").(*model.Auth)

	id, err := utility.ToInt32(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error(err.Error())
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	request := new(model.UserDelete)
	request.ID = id

	if err := u.UserService.Delete(r.Context(), request, auth); err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, "User deleted successfully")
}
