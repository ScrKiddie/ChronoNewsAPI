package test

import (
	"bytes"
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
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})

	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	assert.NoError(t, err, "Failed to create dummy PNG")
	return &buf
}

func TestFileEndpoints(t *testing.T) {
	ts := httptest.NewServer(testRouter)
	defer ts.Close()

	clearTables(testDB)

	appConfig := NewTestConfig()

	adminToken, err := getAuthToken(testDB, ts.URL, "admin@test.com", "admin")
	assert.NoError(t, err, "Failed to get admin token")

	t.Run("Upload Image", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)

		dummyPNGReader := createDummyPNG(t)

		fw, err := w.CreateFormFile("image", "dummy_image.png")
		assert.NoError(t, err)
		_, err = io.Copy(fw, dummyPNGReader)
		assert.NoError(t, err)
		w.Close()

		req, _ := http.NewRequest("POST", ts.URL+"/api/image", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result struct {
			Data model.ImageUploadResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)
		assert.NotEmpty(t, result.Data.Name, "Image name should not be empty")
		assert.NotZero(t, result.Data.ID, "Image ID should not be zero")

		t.Cleanup(func() {
			filePath := filepath.Join(appConfig.Storage.Post, result.Data.Name)
			err := os.Remove(filePath)
			assert.NoError(t, err, "Failed to delete uploaded test file")
		})
	})
}
