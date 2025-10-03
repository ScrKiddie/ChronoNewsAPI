package test

import (
	"bytes"
	"chrononewsapi/internal/config"
	"chrononewsapi/internal/entity"
	"chrononewsapi/internal/model"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserEndpoints(t *testing.T) {
	ts := httptest.NewServer(testRouter)
	defer ts.Close()

	client := config.NewClient()

	clearTables(testDB)

	adminToken, err := getAuthToken(t, testDB, ts.URL, "admin-user@test.com", "admin")
	assert.NoError(t, err, "Failed to get admin token")

	journalistToken, err := getAuthToken(t, testDB, ts.URL, "journalist-user@test.com", "journalist")
	assert.NoError(t, err, "Failed to get journalist token")

	var adminUser entity.User
	err = testDB.Where("email = ?", "admin-user@test.com").First(&adminUser).Error
	assert.NoError(t, err, "Failed to find admin user after creation")
	assert.NotZero(t, adminUser.ID, "Admin user ID should not be zero")

	var newUserID int32

	t.Run("Login - Wrong Credentials", func(t *testing.T) {
		loginData := model.UserLogin{
			Email:        "admin-user@test.com",
			Password:     "WrongPassword!2",
			TokenCaptcha: "Token_Captcha",
		}
		body, err := json.Marshal(loginData)
		assert.NoError(t, err)

		resp, err := client.Post(ts.URL+"/api/user/login", "application/json", bytes.NewBuffer(body))
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Login - Invalid Captcha", func(t *testing.T) {
		originalSecret := appConfig.Captcha.Secret
		appConfig.Captcha.Secret = testConfig.Captcha.Secret.Fail
		t.Cleanup(func() {
			appConfig.Captcha.Secret = originalSecret
		})

		loginData := model.UserLogin{
			Email:        "admin-user@test.com",
			Password:     "Password!23",
			TokenCaptcha: "any-token-will-fail-with-this-secret",
		}
		body, err := json.Marshal(loginData)
		assert.NoError(t, err)

		resp, err := client.Post(ts.URL+"/api/user/login", "application/json", bytes.NewBuffer(body))
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Login - Captcha Already Used", func(t *testing.T) {
		originalSecret := appConfig.Captcha.Secret
		appConfig.Captcha.Secret = testConfig.Captcha.Secret.Usage
		t.Cleanup(func() {
			appConfig.Captcha.Secret = originalSecret
		})

		loginData := model.UserLogin{
			Email:        "admin-user@test.com",
			Password:     "Password!23",
			TokenCaptcha: "Token_Captcha",
		}
		body, err := json.Marshal(loginData)
		assert.NoError(t, err)

		resp, err := client.Post(ts.URL+"/api/user/login", "application/json", bytes.NewBuffer(body))
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Search Users - Empty", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+"/api/user?name=nonexistent", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data []model.UserResponse `json:"data"`
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Len(t, result.Data, 0)
	})

	t.Run("Create User", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("name", "Test User"))
		assert.NoError(t, w.WriteField("email", "testuser@example.com"))
		assert.NoError(t, w.WriteField("phoneNumber", "1234567890"))
		assert.NoError(t, w.WriteField("role", "journalist"))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("POST", ts.URL+"/api/user", &b)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result struct {
			Data model.UserResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, "Test User", result.Data.Name)
		newUserID = result.Data.ID
	})

	t.Run("Create User - Conflict Email", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("name", "Another User"))
		assert.NoError(t, w.WriteField("email", "testuser@example.com"))
		assert.NoError(t, w.WriteField("phoneNumber", "1111111111"))
		assert.NoError(t, w.WriteField("role", "journalist"))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("POST", ts.URL+"/api/user", &b)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("Create User - Conflict Phone Number", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("name", "Another User 2"))
		assert.NoError(t, w.WriteField("email", "anotheruser@example.com"))
		assert.NoError(t, w.WriteField("phoneNumber", "1234567890"))
		assert.NoError(t, w.WriteField("role", "journalist"))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("POST", ts.URL+"/api/user", &b)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("Create User - Bad Request", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("name", ""))
		assert.NoError(t, w.WriteField("email", "bad-request-user"))
		assert.NoError(t, w.WriteField("phoneNumber", "9876543210"))
		assert.NoError(t, w.WriteField("role", "guest"))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("POST", ts.URL+"/api/user", &b)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Create User - Forbidden", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("name", "Forbidden User"))
		assert.NoError(t, w.WriteField("email", "forbidden@example.com"))
		assert.NoError(t, w.WriteField("phoneNumber", "5555555555"))
		assert.NoError(t, w.WriteField("role", "journalist"))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("POST", ts.URL+"/api/user", &b)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+journalistToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Get All Users", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+"/api/user", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data       []model.UserResponse `json:"data"`
			Pagination model.Pagination     `json:"pagination"`
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.NotEmpty(t, result.Data)
		assert.NotZero(t, result.Pagination.TotalItem)
	})

	t.Run("Search Users - Forbidden", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+"/api/user", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+journalistToken)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Get User By ID", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+fmt.Sprintf("/api/user/%d", newUserID), nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data model.UserResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, newUserID, result.Data.ID)
	})

	t.Run("Get User By ID - Not Found", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+"/api/user/99999", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Get User By ID - Forbidden", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+fmt.Sprintf("/api/user/%d", newUserID), nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+journalistToken)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Get User By ID - Cannot Get Self", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+fmt.Sprintf("/api/user/%d", adminUser.ID), nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Update User", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("name", "Updated Test User"))
		assert.NoError(t, w.WriteField("email", "updateduser@example.com"))
		assert.NoError(t, w.WriteField("phoneNumber", "0987654321"))
		assert.NoError(t, w.WriteField("role", "admin"))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/api/user/%d", newUserID), &b)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data model.UserResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Test User", result.Data.Name)
	})

	t.Run("Update User - Admin Deletes Profile Picture", func(t *testing.T) {
		var bCreate bytes.Buffer
		wCreate := multipart.NewWriter(&bCreate)
		dummyPNG := createDummyPNG(t)
		fw, err := wCreate.CreateFormFile("profilePicture", "pic_to_delete.png")
		assert.NoError(t, err)
		_, err = io.Copy(fw, dummyPNG)
		assert.NoError(t, err)
		assert.NoError(t, wCreate.WriteField("name", "User Pic Delete"))
		assert.NoError(t, wCreate.WriteField("email", "user.pic.delete@example.com"))
		assert.NoError(t, wCreate.WriteField("phoneNumber", "1010101010"))
		assert.NoError(t, wCreate.WriteField("role", "journalist"))
		assert.NoError(t, wCreate.Close())

		reqCreate, err := http.NewRequest("POST", ts.URL+"/api/user", &bCreate)
		assert.NoError(t, err)
		reqCreate.Header.Set("Content-Type", wCreate.FormDataContentType())
		reqCreate.Header.Set("Authorization", "Bearer "+adminToken)
		respCreate, err := client.Do(reqCreate)
		assert.NoError(t, err)
		defer func() {
			err := respCreate.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusCreated, respCreate.StatusCode)

		var createdUserResponse struct{ Data model.UserResponse }
		err = json.NewDecoder(respCreate.Body).Decode(&createdUserResponse)
		assert.NoError(t, err)
		userIDToDeletePic := createdUserResponse.Data.ID
		oldPictureName := createdUserResponse.Data.ProfilePicture
		assert.NotEmpty(t, oldPictureName)

		createdFilePath := filepath.Join(appConfig.Storage.Profile, oldPictureName)
		_, err = os.Stat(createdFilePath)
		assert.NoError(t, err, "Profile picture file should exist in storage after creation")

		var bUpdate bytes.Buffer
		wUpdate := multipart.NewWriter(&bUpdate)
		assert.NoError(t, wUpdate.WriteField("name", "User Pic Delete"))
		assert.NoError(t, wUpdate.WriteField("email", "user.pic.delete@example.com"))
		assert.NoError(t, wUpdate.WriteField("phoneNumber", "1010101010"))
		assert.NoError(t, wUpdate.WriteField("role", "journalist"))
		assert.NoError(t, wUpdate.WriteField("deleteProfilePicture", "true"))
		assert.NoError(t, wUpdate.Close())

		reqUpdate, err := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/api/user/%d", userIDToDeletePic), &bUpdate)
		assert.NoError(t, err)
		reqUpdate.Header.Set("Content-Type", wUpdate.FormDataContentType())
		reqUpdate.Header.Set("Authorization", "Bearer "+adminToken)
		respUpdate, err := client.Do(reqUpdate)
		assert.NoError(t, err)
		defer func() {
			err := respUpdate.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusOK, respUpdate.StatusCode)

		var updatedUser entity.User
		testDB.First(&updatedUser, userIDToDeletePic)
		assert.Empty(t, updatedUser.ProfilePicture, "ProfilePicture field in DB should be empty")

		filePath := filepath.Join(appConfig.Storage.Profile, oldPictureName)
		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err), "Old profile picture file should be deleted from storage")
	})

	t.Run("Update User - Admin Replaces Profile Picture", func(t *testing.T) {
		var userToUpdate entity.User
		err = testDB.Where("email = ?", "user.pic.delete@example.com").First(&userToUpdate).Error
		assert.NoError(t, err)

		oldPictureName, err := updateUserProfileByAdmin(t, client, ts.URL, adminToken, &userToUpdate, "old_pic.png")
		assert.NoError(t, err)
		assert.NotEmpty(t, oldPictureName, "First picture name should not be empty")

		oldFilePath := filepath.Join(appConfig.Storage.Profile, oldPictureName)
		_, err = os.Stat(oldFilePath)
		assert.NoError(t, err, "First profile picture should exist before replacement")

		newPictureName, err := updateUserProfileByAdmin(t, client, ts.URL, adminToken, &userToUpdate, "new_pic.png")
		assert.NoError(t, err)
		assert.NotEmpty(t, newPictureName, "New picture name should not be empty")
		assert.NotEqual(t, oldPictureName, newPictureName)

		_, err = os.Stat(oldFilePath)
		assert.True(t, os.IsNotExist(err), "Old profile picture file should be deleted after replacement")
		newFilePath := filepath.Join(appConfig.Storage.Profile, newPictureName)
		_, err = os.Stat(newFilePath)
		assert.NoError(t, err, "New profile picture file should exist after replacement")
	})

	t.Run("Update User - Conflict Email", func(t *testing.T) {
		var bCreate bytes.Buffer
		wCreate := multipart.NewWriter(&bCreate)
		assert.NoError(t, wCreate.WriteField("name", "Temp User"))
		assert.NoError(t, wCreate.WriteField("email", "temp.user@example.com"))
		assert.NoError(t, wCreate.WriteField("phoneNumber", "555444333"))
		assert.NoError(t, wCreate.WriteField("role", "journalist"))
		assert.NoError(t, wCreate.Close())
		reqCreate, err := http.NewRequest("POST", ts.URL+"/api/user", &bCreate)
		assert.NoError(t, err)
		reqCreate.Header.Set("Content-Type", wCreate.FormDataContentType())
		reqCreate.Header.Set("Authorization", "Bearer "+adminToken)
		respCreate, err := client.Do(reqCreate)
		assert.NoError(t, err)
		defer func() {
			err := respCreate.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusCreated, respCreate.StatusCode)

		var bUpdate bytes.Buffer
		wUpdate := multipart.NewWriter(&bUpdate)
		assert.NoError(t, wUpdate.WriteField("name", "Updated Test User"))
		assert.NoError(t, wUpdate.WriteField("email", "temp.user@example.com"))
		assert.NoError(t, wUpdate.WriteField("phoneNumber", "0987654321"))
		assert.NoError(t, wUpdate.WriteField("role", "admin"))
		assert.NoError(t, wUpdate.Close())

		reqUpdate, err := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/api/user/%d", newUserID), &bUpdate)
		assert.NoError(t, err)
		reqUpdate.Header.Set("Content-Type", wUpdate.FormDataContentType())
		reqUpdate.Header.Set("Authorization", "Bearer "+adminToken)
		respUpdate, err := client.Do(reqUpdate)
		assert.NoError(t, err)
		defer func() {
			err := respUpdate.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusConflict, respUpdate.StatusCode)
	})

	t.Run("Update User - Conflict Phone Number", func(t *testing.T) {
		var bCreate bytes.Buffer
		wCreate := multipart.NewWriter(&bCreate)
		assert.NoError(t, wCreate.WriteField("name", "Temp User Phone"))
		assert.NoError(t, wCreate.WriteField("email", "temp.phone@example.com"))
		assert.NoError(t, wCreate.WriteField("phoneNumber", "999888777"))
		assert.NoError(t, wCreate.WriteField("role", "journalist"))
		assert.NoError(t, wCreate.Close())
		reqCreate, err := http.NewRequest("POST", ts.URL+"/api/user", &bCreate)
		assert.NoError(t, err)
		reqCreate.Header.Set("Content-Type", wCreate.FormDataContentType())
		reqCreate.Header.Set("Authorization", "Bearer "+adminToken)
		respCreate, err := client.Do(reqCreate)
		assert.NoError(t, err)
		defer func() {
			err := respCreate.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusCreated, respCreate.StatusCode)

		var bUpdate bytes.Buffer
		wUpdate := multipart.NewWriter(&bUpdate)
		assert.NoError(t, wUpdate.WriteField("name", "Updated Test User"))
		assert.NoError(t, wUpdate.WriteField("email", "updateduser@example.com"))
		assert.NoError(t, wUpdate.WriteField("phoneNumber", "999888777"))
		assert.NoError(t, wUpdate.WriteField("role", "admin"))
		assert.NoError(t, wUpdate.Close())

		reqUpdate, err := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/api/user/%d", newUserID), &bUpdate)
		assert.NoError(t, err)
		reqUpdate.Header.Set("Content-Type", wUpdate.FormDataContentType())
		reqUpdate.Header.Set("Authorization", "Bearer "+adminToken)
		respUpdate, err := client.Do(reqUpdate)
		assert.NoError(t, err)
		defer func() {
			err := respUpdate.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusConflict, respUpdate.StatusCode)
	})

	t.Run("Update User - Forbidden", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("name", "Forbidden Update"))
		assert.NoError(t, w.WriteField("email", "forbidden.update@example.com"))
		assert.NoError(t, w.WriteField("phoneNumber", "111222333"))
		assert.NoError(t, w.WriteField("role", "admin"))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/api/user/%d", newUserID), &b)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+journalistToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Update User - Not Found", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("name", "Non Existent User"))
		assert.NoError(t, w.WriteField("email", "nonexistent@example.com"))
		assert.NoError(t, w.WriteField("phoneNumber", "1231231231"))
		assert.NoError(t, w.WriteField("role", "journalist"))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("PUT", ts.URL+"/api/user/9999", &b)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Get Current User", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+"/api/user/current", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data model.UserResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, "admin-user@test.com", result.Data.Email)
	})

	t.Run("Update Current User Profile", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("name", "Current User Updated"))
		assert.NoError(t, w.WriteField("email", "currentuser.updated@example.com"))
		assert.NoError(t, w.WriteField("phoneNumber", "1122334455"))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("PATCH", ts.URL+"/api/user/current/profile", &b)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Update Current User Profile - Replace Profile Picture", func(t *testing.T) {
		var bUpdate bytes.Buffer
		wUpdate := multipart.NewWriter(&bUpdate)
		assert.NoError(t, wUpdate.WriteField("name", "Current User Updated"))
		assert.NoError(t, wUpdate.WriteField("email", "currentuser.updated@example.com"))
		assert.NoError(t, wUpdate.WriteField("phoneNumber", "1122334455"))
		assert.NoError(t, wUpdate.Close())

		updateReq, err := http.NewRequest("PATCH", ts.URL+"/api/user/current/profile", &bUpdate)
		assert.NoError(t, err)
		updateReq.Header.Set("Content-Type", wUpdate.FormDataContentType())
		updateReq.Header.Set("Authorization", "Bearer "+adminToken)
		updateResp, err := client.Do(updateReq)
		assert.NoError(t, err)
		defer func() {
			err := updateResp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusOK, updateResp.StatusCode)

		var bUpload1 bytes.Buffer
		wUpload1 := multipart.NewWriter(&bUpload1)
		fw1, err := wUpload1.CreateFormFile("profilePicture", "initial_pic.png")
		assert.NoError(t, err)
		_, err = io.Copy(fw1, createDummyPNG(t))
		assert.NoError(t, err)
		assert.NoError(t, wUpload1.WriteField("name", "Current User Updated"))
		assert.NoError(t, wUpload1.WriteField("email", "currentuser.updated@example.com"))
		assert.NoError(t, wUpload1.WriteField("phoneNumber", "1122334455"))
		assert.NoError(t, wUpload1.Close())

		reqUpload1, err := http.NewRequest("PATCH", ts.URL+"/api/user/current/profile", &bUpload1)
		assert.NoError(t, err)
		reqUpload1.Header.Set("Content-Type", wUpload1.FormDataContentType())
		reqUpload1.Header.Set("Authorization", "Bearer "+adminToken)
		resp1, err := client.Do(reqUpload1)
		assert.NoError(t, err)
		defer func() {
			err := resp1.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusOK, resp1.StatusCode)
		var userResp1 struct{ Data model.UserResponse }
		err = json.NewDecoder(resp1.Body).Decode(&userResp1)
		assert.NoError(t, err)
		oldPictureName := userResp1.Data.ProfilePicture
		assert.NotEmpty(t, oldPictureName, "Initial picture name should not be empty")

		oldFilePath := filepath.Join(appConfig.Storage.Profile, oldPictureName)
		_, err = os.Stat(oldFilePath)
		assert.NoError(t, err, "Initial profile picture should exist before replacement")

		var currentUser entity.User
		err = testDB.Where("email = ?", "currentuser.updated@example.com").First(&currentUser).Error
		assert.NoError(t, err)

		newPictureName, err := updateCurrentUserProfile(t, client, ts.URL, adminToken, &currentUser, "replacement_pic.png")
		assert.NoError(t, err)
		assert.NotEmpty(t, newPictureName, "New picture name should not be empty")
		assert.NotEqual(t, oldPictureName, newPictureName)

		_, err = os.Stat(oldFilePath)
		assert.True(t, os.IsNotExist(err), "Old profile picture file should be deleted after replacement")
		newFilePath := filepath.Join(appConfig.Storage.Profile, newPictureName)
		_, err = os.Stat(newFilePath)
		assert.NoError(t, err, "New profile picture file should exist after replacement")
	})

	t.Run("Update Current User Profile - Delete Profile Picture", func(t *testing.T) {
		var bUpload bytes.Buffer
		wUpload := multipart.NewWriter(&bUpload)
		dummyPNG := createDummyPNG(t)
		fw, err := wUpload.CreateFormFile("profilePicture", "test_profile.png")
		assert.NoError(t, err)
		_, err = io.Copy(fw, dummyPNG)
		assert.NoError(t, err)
		assert.NoError(t, wUpload.WriteField("name", "User With Picture"))
		assert.NoError(t, wUpload.WriteField("email", "user.with.pic@example.com"))
		assert.NoError(t, wUpload.WriteField("phoneNumber", "555666777"))
		assert.NoError(t, wUpload.Close())

		reqUpload, err := http.NewRequest("PATCH", ts.URL+"/api/user/current/profile", &bUpload)
		assert.NoError(t, err)
		reqUpload.Header.Set("Content-Type", wUpload.FormDataContentType())
		reqUpload.Header.Set("Authorization", "Bearer "+adminToken)
		respUpload, err := client.Do(reqUpload)
		assert.NoError(t, err)
		defer func() {
			err := respUpload.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusOK, respUpload.StatusCode)

		var uploadResp struct{ Data model.UserResponse }
		err = json.NewDecoder(respUpload.Body).Decode(&uploadResp)
		assert.NoError(t, err)
		pictureNameToDelete := uploadResp.Data.ProfilePicture
		assert.NotEmpty(t, pictureNameToDelete, "Picture name should not be empty after upload")

		filePath := filepath.Join(appConfig.Storage.Profile, pictureNameToDelete)
		_, err = os.Stat(filePath)
		assert.NoError(t, err, "Profile picture file should exist in storage before deletion")

		var bDelete bytes.Buffer
		wDelete := multipart.NewWriter(&bDelete)
		assert.NoError(t, wDelete.WriteField("name", "User With Picture"))
		assert.NoError(t, wDelete.WriteField("email", "user.with.pic@example.com"))
		assert.NoError(t, wDelete.WriteField("phoneNumber", "555666777"))
		assert.NoError(t, wDelete.WriteField("deleteProfilePicture", "true"))
		assert.NoError(t, wDelete.Close())

		reqDelete, err := http.NewRequest("PATCH", ts.URL+"/api/user/current/profile", &bDelete)
		assert.NoError(t, err)
		reqDelete.Header.Set("Content-Type", wDelete.FormDataContentType())
		reqDelete.Header.Set("Authorization", "Bearer "+adminToken)
		respDelete, err := client.Do(reqDelete)
		assert.NoError(t, err)
		defer func() {
			err := respDelete.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusOK, respDelete.StatusCode)

		var updatedUser entity.User
		err = testDB.Where("email = ?", "user.with.pic@example.com").First(&updatedUser).Error
		assert.NoError(t, err)
		assert.Empty(t, updatedUser.ProfilePicture, "ProfilePicture field in DB should be empty after deletion")

		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err), "Profile picture file should be deleted from storage")
	})

	t.Run("Update Current User Password", func(t *testing.T) {
		passwordData := model.UserUpdatePassword{
			OldPassword:     "Password!23",
			Password:        "Password!234",
			ConfirmPassword: "Password!234",
		}
		body, err := json.Marshal(passwordData)
		assert.NoError(t, err)
		req, err := http.NewRequest("PATCH", ts.URL+"/api/user/current/password", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("Update Current User Password - Wrong Old Password", func(t *testing.T) {
		passwordData := model.UserUpdatePassword{
			OldPassword:     "WrongPassword!2",
			Password:        "NewPassword123!",
			ConfirmPassword: "NewPassword123!",
		}
		body, err := json.Marshal(passwordData)
		assert.NoError(t, err)
		req, err := http.NewRequest("PATCH", ts.URL+"/api/user/current/password", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Delete User - Cannot Delete Self", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", ts.URL+fmt.Sprintf("/api/user/%d", adminUser.ID), nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Delete User - In Use", func(t *testing.T) {
		tempCategory := entity.Category{Name: "Temp Category for Deletion Test"}
		err := testDB.Create(&tempCategory).Error
		assert.NoError(t, err)

		post := entity.Post{
			Title:      "Post by user to be deleted",
			Content:    "Some content",
			UserID:     newUserID,
			CategoryID: tempCategory.ID,
		}
		err = testDB.Create(&post).Error
		assert.NoError(t, err)

		req, err := http.NewRequest("DELETE", ts.URL+fmt.Sprintf("/api/user/%d", newUserID), nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		testDB.Where("id = ?", post.ID).Delete(&entity.Post{})
	})

	t.Run("Delete User", func(t *testing.T) {
		var userToDelete entity.User
		err := testDB.First(&userToDelete, newUserID).Error
		assert.NoError(t, err)
		profilePictureName := userToDelete.ProfilePicture

		req, err := http.NewRequest("DELETE", ts.URL+fmt.Sprintf("/api/user/%d", newUserID), nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		if profilePictureName != "" {
			filePath := filepath.Join(appConfig.Storage.Profile, profilePictureName)
			_, err := os.Stat(filePath)
			assert.True(t, os.IsNotExist(err), "Profile picture file should be deleted from storage")
		}
	})

	t.Run("Delete User - Not Found", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", ts.URL+"/api/user/9999", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func updateUserProfileByAdmin(t *testing.T, client *http.Client, serverURL, token string, user *entity.User, fileName string) (string, error) {
	if user == nil {
		return "", fmt.Errorf("user cannot be nil")
	}

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if fileName != "" {
		fw, err := w.CreateFormFile("profilePicture", fileName)
		if err != nil {
			return "", err
		}
		dummyPNG := createDummyPNG(t)
		_, err = io.Copy(fw, dummyPNG)
		if !assert.NoError(t, err) {
			return "", err
		}
	}

	assert.NoError(t, w.WriteField("name", user.Name))
	assert.NoError(t, w.WriteField("email", user.Email))
	assert.NoError(t, w.WriteField("phoneNumber", user.PhoneNumber))
	assert.NoError(t, w.WriteField("role", user.Role))
	if err := w.Close(); err != nil {
		return "", err
	}

	url := serverURL + fmt.Sprintf("/api/user/%d", user.ID)
	req, err := http.NewRequest("PUT", url, &b)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	return executeUserPictureUploadRequest(t, client, req)
}

func updateCurrentUserProfile(t *testing.T, client *http.Client, serverURL, token string, user *entity.User, fileName string) (string, error) {
	if user == nil {
		return "", fmt.Errorf("user cannot be nil")
	}

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if fileName != "" {
		fw, err := w.CreateFormFile("profilePicture", fileName)
		if err != nil {
			return "", err
		}
		dummyPNG := createDummyPNG(t)
		_, err = io.Copy(fw, dummyPNG)
		if !assert.NoError(t, err) {
			return "", err
		}
	}

	assert.NoError(t, w.WriteField("name", user.Name))
	assert.NoError(t, w.WriteField("email", user.Email))
	assert.NoError(t, w.WriteField("phoneNumber", user.PhoneNumber))
	assert.NoError(t, w.Close())

	url := serverURL + "/api/user/current/profile"
	req, err := http.NewRequest("PATCH", url, &b)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	return executeUserPictureUploadRequest(t, client, req)
}

func executeUserPictureUploadRequest(t *testing.T, client *http.Client, req *http.Request) (string, error) {
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		err := resp.Body.Close()
		assert.NoError(t, err)
	}()

	if !assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected OK status for picture upload") {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("request failed with status %s: %s", resp.Status, string(bodyBytes))
	}

	var userResp struct{ Data model.UserResponse }
	err = json.NewDecoder(resp.Body).Decode(&userResp)
	if err != nil {
		return "", err
	}
	return userResp.Data.ProfilePicture, nil
}
