package test

import (
	"bytes"
	"chrononewsapi/internal/entity"
	"chrononewsapi/internal/model"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestResetEndpoints(t *testing.T) {
	ts := httptest.NewServer(testRouter)
	defer ts.Close()

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

	t.Run("Request Password Reset", func(t *testing.T) {
		resetEmailReq := model.ResetEmailRequest{
			Email:        "resetuser@test.com",
			TokenCaptcha: "1x0000000000000000000000000000000AA", // this value is from https://developers.cloudflare.com/turnstile/troubleshooting/testing/
		}
		body, _ := json.Marshal(resetEmailReq)

		req, _ := http.NewRequest("POST", ts.URL+"/api/reset/request", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data interface{} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
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
		body, _ := json.Marshal(resetPassReq)

		req, _ := http.NewRequest("PATCH", ts.URL+"/api/reset", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data interface{} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		loginData := model.UserLogin{
			Email:        "resetuser@test.com",
			Password:     "Password!234",
			TokenCaptcha: "1x0000000000000000000000000000000AA", // this value is from https://developers.cloudflare.com/turnstile/troubleshooting/testing/
		}
		loginBody, _ := json.Marshal(loginData)

		loginResp, err := http.Post(ts.URL+"/api/user/login", "application/json", bytes.NewBuffer(loginBody))
		assert.NoError(t, err)
		defer loginResp.Body.Close()

		assert.Equal(t, http.StatusOK, loginResp.StatusCode, "Login with new password should be successful")

		var loginResult struct {
			Data string `json:"data"`
		}
		err = json.NewDecoder(loginResp.Body).Decode(&loginResult)
		assert.NoError(t, err)
		assert.NotEmpty(t, loginResult.Data, "Token should be present after successful login")
	})
}
