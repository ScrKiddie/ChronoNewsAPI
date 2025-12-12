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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTestCategory(t *testing.T, client *http.Client, token, serverURL string) (int32, error) {
	categoryData := `{"name": "Test Category for Posts"}`
	req, err := http.NewRequest("POST", serverURL+"/api/category", strings.NewReader(categoryData))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to execute request: %v", err)
	}
	defer func() {
		err := resp.Body.Close()
		assert.NoError(t, err)
	}()

	if !assert.Equal(t, http.StatusCreated, resp.StatusCode) {
		return 0, fmt.Errorf("expected status Created; got %v", resp.Status)
	}

	var result struct {
		Data model.CategoryResponse `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return 0, fmt.Errorf("failed to decode response: %v", err)
	}

	return result.Data.ID, err
}

func TestPostEndpoints(t *testing.T) {
	ts := httptest.NewServer(testRouter)
	defer ts.Close()

	clearTables(testDB)

	client := config.NewClient()

	adminToken, err := getAuthToken(t, testDB, ts.URL, "admin-post@test.com", "admin")
	assert.NoError(t, err, "Failed to get admin token")

	journalistToken, err := getAuthToken(t, testDB, ts.URL, "journalist-post@test.com", "journalist")
	assert.NoError(t, err, "Failed to get journalist token")

	var adminUser entity.User
	err = testDB.Where("email = ?", "admin-post@test.com").First(&adminUser).Error
	assert.NoError(t, err, "Failed to find admin user for post tests")

	categoryID, err := createTestCategory(t, client, adminToken, ts.URL)
	assert.NoError(t, err, "Failed to create test category")
	assert.NotZero(t, categoryID, "Category ID should not be zero")

	var newPostID int32
	var journalistPostID int32

	t.Run("Create Post", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)

		dummyPNGReader := createDummyPNG(t)
		fw, err := w.CreateFormFile("thumbnail", "test_thumbnail.png")
		assert.NoError(t, err)
		_, err = io.Copy(fw, dummyPNGReader)
		assert.NoError(t, err)

		assert.NoError(t, w.WriteField("title", "Test Post"))
		assert.NoError(t, w.WriteField("summary", "This is a test post."))
		assert.NoError(t, w.WriteField("content", "<p>This is the full content of the test post.</p>"))
		assert.NoError(t, w.WriteField("categoryID", fmt.Sprintf("%d", categoryID)))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("POST", ts.URL+"/api/post", &b)
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
			Data model.PostResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, "Test Post", result.Data.Title)
		assert.NotEmpty(t, result.Data.Thumbnail, "Thumbnail should be present in response")
		newPostID = result.Data.ID

		fileName := filepath.Base(result.Data.Thumbnail)

		filePath := filepath.Join(appConfig.Storage.Post, fileName)
		_, err = os.Stat(filePath)
		assert.NoError(t, err, "Thumbnail file should exist in storage after post creation")
	})

	t.Run("Create Post - As Journalist", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("title", "Journalist Post"))
		assert.NoError(t, w.WriteField("summary", "This is a post by a journalist."))
		assert.NoError(t, w.WriteField("content", "<p>Content by journalist.</p>"))
		assert.NoError(t, w.WriteField("categoryID", fmt.Sprintf("%d", categoryID)))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("POST", ts.URL+"/api/post", &b)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+journalistToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result struct {
			Data model.PostResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, "Journalist Post", result.Data.Title)
		journalistPostID = result.Data.ID
	})

	t.Run("Create Post - Bad Request", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("title", ""))
		assert.NoError(t, w.WriteField("summary", "This is a bad post."))
		assert.NoError(t, w.WriteField("content", "<p>This content should not be saved.</p>"))
		assert.NoError(t, w.WriteField("categoryID", fmt.Sprintf("%d", categoryID)))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("POST", ts.URL+"/api/post", &b)
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

	t.Run("Create Post - Category Not Found", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("title", "Post with invalid category"))
		assert.NoError(t, w.WriteField("summary", "Summary"))
		assert.NoError(t, w.WriteField("content", "<p>Content</p>"))
		assert.NoError(t, w.WriteField("categoryID", "99999"))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("POST", ts.URL+"/api/post", &b)
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

	t.Run("Create Post - Unauthenticated", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("title", "Unauthenticated Post"))
		assert.NoError(t, w.WriteField("categoryID", fmt.Sprintf("%d", categoryID)))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("POST", ts.URL+"/api/post", &b)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", w.FormDataContentType())

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Get All Posts", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+"/api/post", nil)
		assert.NoError(t, err)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data       []model.PostResponseWithPreload `json:"data"`
			Pagination model.Pagination                `json:"pagination"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.NotEmpty(t, result.Data)
		assert.NotZero(t, result.Pagination.TotalItem)
	})

	t.Run("Search Posts - No Result", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+"/api/post?title=nonexistentposttitle", nil)
		assert.NoError(t, err)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data       []model.PostResponseWithPreload `json:"data"`
			Pagination model.Pagination                `json:"pagination"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Empty(t, result.Data)
		assert.Zero(t, result.Pagination.TotalItem)
	})

	t.Run("Get Post By ID", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+fmt.Sprintf("/api/post/%d", newPostID), nil)
		assert.NoError(t, err)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data model.PostResponseWithPreload `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, newPostID, result.Data.ID)
	})

	t.Run("Get Post By ID - Not Found", func(t *testing.T) {
		req, err := http.NewRequest("GET", ts.URL+"/api/post/99999", nil)
		assert.NoError(t, err)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Update Post", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("title", "Updated Test Post"))
		assert.NoError(t, w.WriteField("summary", "This is an updated test post."))
		assert.NoError(t, w.WriteField("content", "<p>This is the updated full content.</p>"))
		assert.NoError(t, w.WriteField("categoryID", fmt.Sprintf("%d", categoryID)))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/api/post/%d", newPostID), &b)
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
			Data model.PostResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Test Post", result.Data.Title)
	})

	t.Run("Update Post - Delete Thumbnail", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("title", "Post Without Thumbnail"))
		assert.NoError(t, w.WriteField("summary", "Summary for post without thumbnail"))
		assert.NoError(t, w.WriteField("content", "<p>Content for post without thumbnail</p>"))
		assert.NoError(t, w.WriteField("categoryID", fmt.Sprintf("%d", categoryID)))
		assert.NoError(t, w.WriteField("deleteThumbnail", "true"))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/api/post/%d", newPostID), &b)
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
			Data model.PostResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Empty(t, result.Data.Thumbnail, "Thumbnail should be empty after deletion")
	})

	t.Run("Update Post - Admin Changes Author", func(t *testing.T) {
		var journalistUser entity.User
		err := testDB.Where("email = ?", "journalist-post@test.com").First(&journalistUser).Error
		assert.NoError(t, err)

		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("title", "Reassigned Post"))
		assert.NoError(t, w.WriteField("summary", "This post is now owned by a journalist."))
		assert.NoError(t, w.WriteField("content", "<p>Reassigned content.</p>"))
		assert.NoError(t, w.WriteField("categoryID", fmt.Sprintf("%d", categoryID)))
		assert.NoError(t, w.WriteField("userID", fmt.Sprintf("%d", journalistUser.ID)))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/api/post/%d", newPostID), &b)
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
			Data model.PostResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, "Reassigned Post", result.Data.Title)
		assert.Equal(t, journalistUser.ID, result.Data.UserID, "Post should be reassigned to the journalist")
	})

	t.Run("Update Post - Admin Assigns to Non-existent User", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("title", "Post to non-existent user"))
		assert.NoError(t, w.WriteField("summary", "Summary for non-existent user post"))
		assert.NoError(t, w.WriteField("content", "<p>Content for non-existent user post</p>"))
		assert.NoError(t, w.WriteField("categoryID", fmt.Sprintf("%d", categoryID)))
		assert.NoError(t, w.WriteField("userID", "99999"))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/api/post/%d", newPostID), &b)
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

	t.Run("Update Post - As Journalist (Owner)", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("title", "Updated by Journalist"))
		assert.NoError(t, w.WriteField("summary", "This is an updated journalist post."))
		assert.NoError(t, w.WriteField("content", "<p>Updated content by journalist.</p>"))
		assert.NoError(t, w.WriteField("categoryID", fmt.Sprintf("%d", categoryID)))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/api/post/%d", journalistPostID), &b)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+journalistToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Update Post - As Journalist (Not Owner)", func(t *testing.T) {
		var adminOwnedPost entity.Post
		err := testDB.Create(&entity.Post{Title: "Admin Only Post", UserID: adminUser.ID, CategoryID: categoryID, Summary: "s", Content: "c"}).Scan(&adminOwnedPost).Error
		assert.NoError(t, err)

		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.WriteField("title", "Forbidden Update"))
		assert.NoError(t, w.WriteField("summary", "This update should fail."))
		assert.NoError(t, w.WriteField("content", "<p>Forbidden content.</p>"))
		assert.NoError(t, w.WriteField("categoryID", fmt.Sprintf("%d", categoryID)))
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/api/post/%d", adminOwnedPost.ID), &b)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+journalistToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Increment Post View", func(t *testing.T) {
		req, err := http.NewRequest("PATCH", ts.URL+fmt.Sprintf("/api/post/%d/view", newPostID), nil)
		assert.NoError(t, err)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var post entity.Post
		err = testDB.First(&post, newPostID).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(1), post.ViewCount)
	})

	t.Run("Increment Post View - Not Found", func(t *testing.T) {
		req, err := http.NewRequest("PATCH", ts.URL+"/api/post/99999/view", nil)
		assert.NoError(t, err)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Delete Post", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", ts.URL+fmt.Sprintf("/api/post/%d", newPostID), nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var fileCount int64
		testDB.Model(&entity.File{}).Where("post_id = ?", newPostID).Count(&fileCount)
		assert.Zero(t, fileCount, "File records associated with the post should be deleted from the database")
	})

	t.Run("Delete Post - As Journalist (Owner)", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", ts.URL+fmt.Sprintf("/api/post/%d", journalistPostID), nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+journalistToken)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Delete Post - As Journalist (Not Owner)", func(t *testing.T) {
		adminOwnedPost := entity.Post{Title: "Temp Delete Post", UserID: adminUser.ID, CategoryID: categoryID, Summary: "s", Content: "c"}
		err := testDB.Create(&adminOwnedPost).Error
		assert.NoError(t, err)

		reqDel, err := http.NewRequest("DELETE", ts.URL+fmt.Sprintf("/api/post/%d", adminOwnedPost.ID), nil)
		assert.NoError(t, err)
		reqDel.Header.Set("Authorization", "Bearer "+journalistToken)

		respDel, err := client.Do(reqDel)
		assert.NoError(t, err)
		defer func() {
			err := respDel.Body.Close()
			assert.NoError(t, err)
		}()
		assert.Equal(t, http.StatusNotFound, respDel.StatusCode)
	})

	t.Run("Delete Post - ID Not Found", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", ts.URL+"/api/post/99999", nil)
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
