package test

import (
	"chrononewsapi/internal/config"
	"chrononewsapi/internal/entity"
	"chrononewsapi/internal/model"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCategoryEndpoints(t *testing.T) {
	ts := httptest.NewServer(testRouter)
	defer ts.Close()

	client := config.NewClient()

	clearTables(testDB)

	adminToken, err := getAuthToken(t, testDB, ts.URL, "admin-category@test.com", "admin")
	assert.NoError(t, err, "Failed to get admin token")

	journalistToken, err := getAuthToken(t, testDB, ts.URL, "journalist-category@test.com", "journalist")
	assert.NoError(t, err, "Failed to get journalist token")

	var newCategoryID int32

	t.Run("Get All Categories - Empty", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+"/api/category", nil)
		assert.NoError(t, err)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data []model.CategoryResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Len(t, result.Data, 0)
	})

	t.Run("Create Category", func(t *testing.T) {
		categoryData := `{"name": "Test Category"}`
		req, err := http.NewRequest("POST", ts.URL+"/api/category", strings.NewReader(categoryData))
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

		var result struct {
			Data model.CategoryResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, "Test Category", result.Data.Name)
		newCategoryID = result.Data.ID
	})

	t.Run("Create Category - Conflict", func(t *testing.T) {
		categoryData := `{"name": "Test Category"}`
		req, err := http.NewRequest("POST", ts.URL+"/api/category", strings.NewReader(categoryData))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("Create Category - Bad Request", func(t *testing.T) {
		categoryData := `{"name": ""}`
		req, err := http.NewRequest("POST", ts.URL+"/api/category", strings.NewReader(categoryData))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Create Category - Forbidden", func(t *testing.T) {
		categoryData := `{"name": "Forbidden Category"}`
		req, err := http.NewRequest("POST", ts.URL+"/api/category", strings.NewReader(categoryData))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+journalistToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Get All Categories", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+"/api/category", nil)
		assert.NoError(t, err)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data []model.CategoryResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.NotEmpty(t, result.Data)
	})

	t.Run("Get Category By ID", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+fmt.Sprintf("/api/category/%d", newCategoryID), nil)
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
			Data model.CategoryResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, newCategoryID, result.Data.ID)
	})

	t.Run("Get Category By ID - Not Found", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+"/api/category/9999", nil)
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

	t.Run("Get Category By ID - Forbidden", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+fmt.Sprintf("/api/category/%d", newCategoryID), nil)
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

	t.Run("Update Category", func(t *testing.T) {
		updateData := `{"name": "Updated Test Category"}`
		req, err := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/api/category/%d", newCategoryID), strings.NewReader(updateData))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data model.CategoryResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Test Category", result.Data.Name)
	})

	t.Run("Update Category - Conflict", func(t *testing.T) {
		conflictCategoryData := `{"name": "Conflict Category"}`
		reqCreate, err := http.NewRequest("POST", ts.URL+"/api/category", strings.NewReader(conflictCategoryData))
		assert.NoError(t, err)
		reqCreate.Header.Set("Content-Type", "application/json")
		reqCreate.Header.Set("Authorization", "Bearer "+adminToken)
		respCreate, err := client.Do(reqCreate)
		assert.NoError(t, err)
		defer func() {
			err := respCreate.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusCreated, respCreate.StatusCode)

		updateData := `{"name": "Conflict Category"}`
		reqUpdate, err := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/api/category/%d", newCategoryID), strings.NewReader(updateData))
		assert.NoError(t, err)
		reqUpdate.Header.Set("Content-Type", "application/json")
		reqUpdate.Header.Set("Authorization", "Bearer "+adminToken)

		respUpdate, err := client.Do(reqUpdate)
		assert.NoError(t, err)
		defer func() {
			err := respUpdate.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusConflict, respUpdate.StatusCode)
	})

	t.Run("Update Category - Not Found", func(t *testing.T) {
		updateData := `{"name": "Non Existent"}`
		req, err := http.NewRequest("PUT", ts.URL+"/api/category/9999", strings.NewReader(updateData))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Update Category - Forbidden", func(t *testing.T) {
		updateData := `{"name": "Forbidden Update"}`
		req, err := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/api/category/%d", newCategoryID), strings.NewReader(updateData))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+journalistToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Delete Category - In Use", func(t *testing.T) {
		var user model.UserResponse
		err := testDB.Model(&entity.User{}).Where("email = ?", "admin-category@test.com").First(&user).Error
		assert.NoError(t, err)

		post := entity.Post{
			Title:      "Post using category",
			Content:    "Some content",
			UserID:     user.ID,
			CategoryID: newCategoryID,
		}
		err = testDB.Create(&post).Error
		assert.NoError(t, err)

		req, err := http.NewRequest("DELETE", ts.URL+fmt.Sprintf("/api/category/%d", newCategoryID), nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		result := testDB.Where("id = ?", post.ID).Delete(&entity.Post{})
		assert.NoError(t, result.Error)
	})

	t.Run("Delete Category", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", ts.URL+fmt.Sprintf("/api/category/%d", newCategoryID), nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Delete Category - Not Found", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", ts.URL+"/api/category/9999", nil)
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

	t.Run("Delete Category - Forbidden", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", ts.URL+"/api/category/9998", nil)
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
}
