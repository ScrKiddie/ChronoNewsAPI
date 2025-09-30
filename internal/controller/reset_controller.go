package controller

import (
	"chrononewsapi/internal/model"
	"chrononewsapi/internal/service"
	"chrononewsapi/internal/utility"
	"encoding/json"
	"log/slog"
	"net/http"
)

type ResetController struct {
	ResetService *service.ResetService
}

func NewResetController(resetService *service.ResetService) *ResetController {
	return &ResetController{ResetService: resetService}
}

// RequestResetEmail handles the request to send a password reset email
// @Summary Request password reset
// @Description Sends an email containing a password reset link
// @Tags Reset
// @Accept json
// @Produce json
// @Param resetEmail body model.ResetEmailRequest true "User email for password reset"
// @Success 200 {object} utility.ResponseSuccess
// @Failure 400 {object} utility.ResponseError
// @Failure 404 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/reset/email [post]
func (c *ResetController) RequestResetEmail(w http.ResponseWriter, r *http.Request) {
	request := new(model.ResetEmailRequest)
	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		slog.Error("Failed to decode request reset email", "error", err)
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	if err := c.ResetService.ResetEmail(r.Context(), request); err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, "Reset email has been sent successfully")
}

// Reset handles the password reset process
// @Summary Reset password
// @Description Resets the user's password using the provided reset code and new password
// @Tags Reset
// @Accept json
// @Produce json
// @Param resetRequest body model.ResetRequest true "Request containing reset code and new password"
// @Success 200 {object} utility.ResponseSuccess
// @Failure 400 {object} utility.ResponseError
// @Failure 404 {object} utility.ResponseError
// @Failure 500 {object} utility.ResponseError
// @Router /api/reset/password [post]
func (c *ResetController) Reset(w http.ResponseWriter, r *http.Request) {
	request := new(model.ResetRequest)
	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		slog.Error("Failed to decode reset password request", "error", err)
		utility.CreateErrorResponse(w, utility.ErrBadRequest.Code, utility.ErrBadRequest.Message)
		return
	}

	if err := c.ResetService.Reset(r.Context(), request); err != nil {
		utility.CreateErrorResponse(w, err.(*utility.CustomError).Code, err.(*utility.CustomError).Message)
		return
	}

	utility.CreateSuccessResponse(w, http.StatusOK, "Password has been reset successfully")
}
