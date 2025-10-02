package test

import (
	"bytes"
	"chrononewsapi/internal/model"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserEndpoints(t *testing.T) {
	ts := httptest.NewServer(testRouter)
	defer ts.Close()

	clearTables(testDB)

	adminToken, err := getAuthToken(testDB, ts.URL, "admin@test.com", "admin")
	assert.NoError(t, err, "Failed to get admin token")

	var newUserID int32

	t.Run("Create User", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		w.WriteField("name", "Test User")
		w.WriteField("email", "testuser@example.com")
		w.WriteField("phoneNumber", "1234567890")
		w.WriteField("role", "journalist")
		w.Close()

		req, _ := http.NewRequest("POST", ts.URL+"/api/user", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result struct {
			Data model.UserResponse `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "Test User", result.Data.Name)
		newUserID = result.Data.ID
	})

	t.Run("Get All Users", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/user", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data       []model.UserResponse `json:"data"`
			Pagination model.Pagination     `json:"pagination"`
		}

		json.NewDecoder(resp.Body).Decode(&result)
		assert.NotEmpty(t, result.Data)
		assert.NotZero(t, result.Pagination.TotalItem)
	})

	t.Run("Get User By ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+fmt.Sprintf("/api/user/%d", newUserID), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data model.UserResponse `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, newUserID, result.Data.ID)
	})

	t.Run("Update User", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		w.WriteField("name", "Updated Test User")
		w.WriteField("email", "updateduser@example.com")
		w.WriteField("phoneNumber", "0987654321")
		w.WriteField("role", "admin")
		w.Close()

		req, _ := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/api/user/%d", newUserID), &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data model.UserResponse `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "Updated Test User", result.Data.Name)
	})

	t.Run("Get Current User", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/user/current", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data model.UserResponse `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "admin@test.com", result.Data.Email)
	})

	t.Run("Update Current User Profile", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		w.WriteField("name", "Current User Updated")
		w.WriteField("email", "currentuser.updated@example.com")
		w.WriteField("phoneNumber", "1122334455")
		w.Close()

		req, _ := http.NewRequest("PATCH", ts.URL+"/api/user/current/profile", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Update Current User Password", func(t *testing.T) {
		passwordData := model.UserUpdatePassword{
			OldPassword:     "Password!23",
			Password:        "Password!234",
			ConfirmPassword: "Password!234",
		}
		body, _ := json.Marshal(passwordData)
		req, _ := http.NewRequest("PATCH", ts.URL+"/api/user/current/password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("Delete User", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", ts.URL+fmt.Sprintf("/api/user/%d", newUserID), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
