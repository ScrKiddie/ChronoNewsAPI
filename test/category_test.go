package test

import (
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

	clearTables(testDB)

	token, err := getAuthToken(testDB, ts.URL, "journalist@test.com", "admin")
	assert.NoError(t, err, "Failed to get auth token")

	var newCategoryID int32

	t.Run("Create Category", func(t *testing.T) {
		categoryData := `{"name": "Test Category"}`
		req, _ := http.NewRequest("POST", ts.URL+"/api/category", strings.NewReader(categoryData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result struct {
			Data model.CategoryResponse `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "Test Category", result.Data.Name)
		newCategoryID = result.Data.ID
	})

	t.Run("Get All Categories", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/category", nil)
		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data []model.CategoryResponse `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.NotEmpty(t, result.Data)
	})

	t.Run("Get Category By ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+fmt.Sprintf("/api/category/%d", newCategoryID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data model.CategoryResponse `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, newCategoryID, result.Data.ID)
	})

	t.Run("Update Category", func(t *testing.T) {
		updateData := `{"name": "Updated Test Category"}`
		req, _ := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/api/category/%d", newCategoryID), strings.NewReader(updateData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data model.CategoryResponse `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "Updated Test Category", result.Data.Name)
	})

	t.Run("Delete Category", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", ts.URL+fmt.Sprintf("/api/category/%d", newCategoryID), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
