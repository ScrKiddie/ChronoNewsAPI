package test

import (
	"bytes"
	"chrononewsapi/internal/config"
	"chrononewsapi/internal/entity"
	"chrononewsapi/internal/model"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestResetEndpoints(t *testing.T) {
	ts := httptest.NewServer(testRouter)
	defer ts.Close()

	client := config.NewClient()

	clearTables(testDB)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password!23"), bcrypt.DefaultCost)
	user := entity.User{
		Name:     "Reset User",
		Email:    "resetuser@test.com",
		Password: string(hashedPassword),
		Role:     "journalist",
	}
	err := testDB.Create(&user).Error
	assert.NoError(t, err, "Failed to create user for reset test")

	t.Run("Request Password Reset - Invalid Email", func(t *testing.T) {
		resetEmailReq := model.ResetEmailRequest{
			Email:        "invalid-email",
			TokenCaptcha: "Token_Captcha",
		}
		body, err := json.Marshal(resetEmailReq)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", ts.URL+"/api/reset/request", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Request Password Reset - Invalid Captcha", func(t *testing.T) {
		originalSecret := appConfig.Captcha.Secret
		appConfig.Captcha.Secret = testConfig.Captcha.Secret.Fail
		t.Cleanup(func() {
			appConfig.Captcha.Secret = originalSecret
		})

		resetEmailReq := model.ResetEmailRequest{
			Email:        "resetuser@test.com",
			TokenCaptcha: "any-token-will-fail-with-this-secret",
		}
		body, err := json.Marshal(resetEmailReq)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", ts.URL+"/api/reset/request", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Request Password Reset - Captcha Already Used", func(t *testing.T) {
		originalSecret := appConfig.Captcha.Secret
		appConfig.Captcha.Secret = testConfig.Captcha.Secret.Usage
		t.Cleanup(func() {
			appConfig.Captcha.Secret = originalSecret
		})

		resetEmailReq := model.ResetEmailRequest{
			Email:        "resetuser@test.com",
			TokenCaptcha: "Token_Captcha",
		}
		body, err := json.Marshal(resetEmailReq)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", ts.URL+"/api/reset/request", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Request Password Reset - User Not Found", func(t *testing.T) {
		resetEmailReq := model.ResetEmailRequest{
			Email:        "nonexistent@test.com",
			TokenCaptcha: "Token_Captcha",
		}
		body, err := json.Marshal(resetEmailReq)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", ts.URL+"/api/reset/request", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var resetData entity.Reset
		err = testDB.First(&resetData, "user_id = ?", 99999).Error
		assert.Error(t, err)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Should not find a reset token for a non-existent user")
	})

	t.Run("Request Password Reset", func(t *testing.T) {
		resetEmailReq := model.ResetEmailRequest{
			Email:        "resetuser@test.com",
			TokenCaptcha: "Token_Captcha",
		}
		body, err := json.Marshal(resetEmailReq)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", ts.URL+"/api/reset/request", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data interface{} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
	})

	t.Run("Request Password Reset - Update Existing Token", func(t *testing.T) {
		var firstReset entity.Reset
		err := testDB.First(&firstReset, "user_id = ?", user.ID).Error
		assert.NoError(t, err, "Should find the first reset token")
		firstCode := firstReset.Code

		resetEmailReq := model.ResetEmailRequest{
			Email:        "resetuser@test.com",
			TokenCaptcha: "Token_Captcha",
		}
		body, err := json.Marshal(resetEmailReq)
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", ts.URL+"/api/reset/request", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var count int64
		testDB.Model(&entity.Reset{}).Where("user_id = ?", user.ID).Count(&count)
		assert.Equal(t, int64(1), count, "There should still be only one reset token for the user")

		var secondReset entity.Reset
		err = testDB.First(&secondReset, "user_id = ?", user.ID).Error
		assert.NoError(t, err)
		assert.NotEqual(t, firstCode, secondReset.Code, "The reset code should have been updated")
	})

	t.Run("Reset Password - Invalid Code", func(t *testing.T) {
		resetPassReq := model.ResetRequest{
			Code:            "invalid-reset-code",
			Password:        "NewPassword!234",
			ConfirmPassword: "NewPassword!234",
		}
		body, err := json.Marshal(resetPassReq)
		assert.NoError(t, err)

		req, err := http.NewRequest("PATCH", ts.URL+"/api/reset", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Reset Password - Expired Code", func(t *testing.T) {
		expiredUser := entity.User{
			Name:     "Expired User",
			Email:    "expireduser@test.com",
			Password: string(hashedPassword),
			Role:     "journalist",
		}
		err := testDB.Create(&expiredUser).Error
		assert.NoError(t, err, "Failed to create expired user for reset test")

		expiredReset := entity.Reset{
			UserID:    expiredUser.ID,
			Code:      "expired-code-123",
			ExpiredAt: time.Now().Add(-time.Hour).Unix(),
		}
		err = testDB.Create(&expiredReset).Error
		assert.NoError(t, err, "Failed to create expired reset token")

		resetPassReq := model.ResetRequest{
			Code:            expiredReset.Code,
			Password:        "NewPassword!234",
			ConfirmPassword: "NewPassword!234",
		}
		body, err := json.Marshal(resetPassReq)
		assert.NoError(t, err)

		req, err := http.NewRequest("PATCH", ts.URL+"/api/reset", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var deletedReset entity.Reset
		err = testDB.First(&deletedReset, "code = ?", expiredReset.Code).Error
		assert.Error(t, err, "Expired token should be deleted after a failed attempt")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("Reset Password - Password Mismatch/Invalid", func(t *testing.T) {
		resetPassReq := model.ResetRequest{
			Code:            "some-valid-code",
			Password:        "short",
			ConfirmPassword: "mismatch",
		}
		body, err := json.Marshal(resetPassReq)
		assert.NoError(t, err)

		req, err := http.NewRequest("PATCH", ts.URL+"/api/reset", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Reset Password", func(t *testing.T) {
		var resetData entity.Reset
		err := testDB.First(&resetData, "user_id = ?", user.ID).Error
		assert.NoError(t, err, "Failed to find reset code in DB")

		resetPassReq := model.ResetRequest{
			Code:            resetData.Code,
			Password:        "Password!234",
			ConfirmPassword: "Password!234",
		}
		body, err := json.Marshal(resetPassReq)
		assert.NoError(t, err)

		req, err := http.NewRequest("PATCH", ts.URL+"/api/reset", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data interface{} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		loginData := model.UserLogin{
			Email:        "resetuser@test.com",
			Password:     "Password!234",
			TokenCaptcha: "Token_Captcha",
		}
		loginBody, err := json.Marshal(loginData)
		assert.NoError(t, err)

		loginResp, err := client.Post(ts.URL+"/api/user/login", "application/json", bytes.NewBuffer(loginBody))
		assert.NoError(t, err)
		defer func() {
			err := loginResp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, loginResp.StatusCode, "Login with new password should be successful")

		var loginResult struct {
			Data string `json:"data"`
		}
		err = json.NewDecoder(loginResp.Body).Decode(&loginResult)
		assert.NoError(t, err)
		assert.NotEmpty(t, loginResult.Data, "Token should be present after successful login")
	})

	t.Run("Reset Password - Verify Token Deletion", func(t *testing.T) {
		var resetData entity.Reset
		err := testDB.First(&resetData, "user_id = ?", user.ID).Error
		assert.Error(t, err, "An error should occur when trying to find a deleted token")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "The error should specifically be 'record not found'")
	})
}
