package test

import (
	"bytes"
	"chrononewsapi/internal/config"
	"chrononewsapi/internal/model"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createDummyPNG(t *testing.T) io.Reader {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})

	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	assert.NoError(t, err, "Failed to create dummy PNG")
	return &buf
}

func TestFileEndpoints(t *testing.T) {
	ts := httptest.NewServer(testRouter)
	defer ts.Close()

	client := config.NewClient()

	clearTables(testDB)

	adminToken, err := getAuthToken(t, testDB, ts.URL, "admin-file@test.com", "admin")
	assert.NoError(t, err, "Failed to get admin token")

	journalistToken, err := getAuthToken(t, testDB, ts.URL, "journalist-file@test.com", "journalist")
	assert.NoError(t, err, "Failed to get journalist token")

	t.Run("Upload Image", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)

		dummyPNGReader := createDummyPNG(t)

		fw, err := w.CreateFormFile("image", "dummy_image.png")
		assert.NoError(t, err)
		_, err = io.Copy(fw, dummyPNGReader)
		assert.NoError(t, err)
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("POST", ts.URL+"/api/image", &b)
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
			Data model.ImageUploadResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.NotEmpty(t, result.Data.Name, "Image name should not be empty")
		assert.NotZero(t, result.Data.ID, "Image ID should not be zero")

		t.Cleanup(func() {

			fileName := filepath.Base(result.Data.Name)

			filePath := filepath.Join(appConfig.Storage.Attachment, fileName)
			err := os.Remove(filePath)
			assert.NoError(t, err, "Failed to delete uploaded test file")
		})
	})

	t.Run("Upload Image - Authenticated Non-Admin", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)

		dummyPNGReader := createDummyPNG(t)

		fw, err := w.CreateFormFile("image", "dummy_image.png")
		assert.NoError(t, err)
		_, err = io.Copy(fw, dummyPNGReader)
		assert.NoError(t, err)
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("POST", ts.URL+"/api/image", &b)
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
	})

	t.Run("Upload Image - No File", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("POST", ts.URL+"/api/image", &b)
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

	t.Run("Upload Image - Invalid File Type", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)

		fw, err := w.CreateFormFile("image", "dummy.txt")
		assert.NoError(t, err)
		_, err = fw.Write([]byte("this is not an image"))
		assert.NoError(t, err)
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("POST", ts.URL+"/api/image", &b)
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

	t.Run("Upload Image - No Token", func(t *testing.T) {
		req, err := http.NewRequest("POST", ts.URL+"/api/image", nil)
		assert.NoError(t, err)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			assert.NoError(t, err)
		}()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Upload Image - File Too Large", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)

		largeFileSize := 3 * 1024 * 1024
		largeFileContent := make([]byte, largeFileSize)

		fw, err := w.CreateFormFile("image", "large_image.jpg")
		assert.NoError(t, err)
		_, err = fw.Write(largeFileContent)
		assert.NoError(t, err)
		assert.NoError(t, w.Close())

		req, err := http.NewRequest("POST", ts.URL+"/api/image", &b)
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
}
