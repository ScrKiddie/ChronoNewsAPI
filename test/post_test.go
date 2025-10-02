package test

import (
	"bytes"
	"chrononewsapi/internal/model"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTestCategory(token, serverURL string) (int32, error) {
	categoryData := `{"name": "Test Category for Posts"}`
	req, _ := http.NewRequest("POST", serverURL+"/api/category", strings.NewReader(categoryData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("expected status Created; got %v", resp.Status)
	}

	var result struct {
		Data model.CategoryResponse `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Data.ID, nil
}

func TestPostEndpoints(t *testing.T) {
	ts := httptest.NewServer(testRouter)
	defer ts.Close()

	clearTables(testDB)

	adminToken, err := getAuthToken(testDB, ts.URL, "admin@test.com", "admin")
	assert.NoError(t, err, "Failed to get admin token")

	categoryID, err := createTestCategory(adminToken, ts.URL)
	assert.NoError(t, err, "Failed to create test category")

	var newPostID int32

	t.Run("Create Post", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		w.WriteField("title", "Test Post")
		w.WriteField("summary", "This is a test post.")
		w.WriteField("content", "<p>This is the full content of the test post.</p>")
		w.WriteField("categoryID", fmt.Sprintf("%d", categoryID))
		w.Close()

		req, _ := http.NewRequest("POST", ts.URL+"/api/post", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result struct {
			Data model.PostResponse `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "Test Post", result.Data.Title)
		newPostID = result.Data.ID
	})

	t.Run("Get All Posts", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/post", nil)
		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data       []model.PostResponseWithPreload `json:"data"`
			Pagination model.Pagination                `json:"pagination"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.NotEmpty(t, result.Data)
		assert.NotZero(t, result.Pagination.TotalItem)
	})

	t.Run("Get Post By ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+fmt.Sprintf("/api/post/%d", newPostID), nil)
		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data model.PostResponseWithPreload `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, newPostID, result.Data.ID)
	})

	t.Run("Update Post", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		w.WriteField("title", "Updated Test Post")
		w.WriteField("summary", "This is an updated test post.")
		w.WriteField("content", "<p>This is the updated full content.</p>")
		w.WriteField("categoryID", fmt.Sprintf("%d", categoryID))
		w.Close()

		req, _ := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/api/post/%d", newPostID), &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data model.PostResponse `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "Updated Test Post", result.Data.Title)
	})

	t.Run("Increment Post View", func(t *testing.T) {
		req, _ := http.NewRequest("PATCH", ts.URL+fmt.Sprintf("/api/post/%d/view", newPostID), nil)
		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Delete Post", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", ts.URL+fmt.Sprintf("/api/post/%d", newPostID), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
