package test

import (
	"bytes"
	"chrononewsapi/internal/bootstrap"
	"chrononewsapi/internal/config"
	"chrononewsapi/internal/entity"
	"chrononewsapi/internal/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	testDB      *gorm.DB
	testRouter  *chi.Mux
	testConfig  *TestConfig
	appConfig   *config.Config
	testTempDir string
)

func convertTestConfigToAppConfig(testCfg *TestConfig) *config.Config {
	return &config.Config{
		Web: config.WebConfig{
			Port:        testCfg.Web.Port,
			CorsOrigins: testCfg.Web.CorsOrigins,
			BaseURL:     fmt.Sprintf("http://localhost:%s", testCfg.Web.Port),

			ClientURL: "http://test-client.com",
			ClientPaths: config.ClientPathConfig{
				Post:     "/post",
				Category: "/category",
				Reset:    "/reset",
				Forgot:   "/forgot",
			},
		},
		DB:      testCfg.DB,
		JWT:     testCfg.JWT,
		Captcha: config.CaptchaConfig{Secret: testCfg.Captcha.Secret.Pass},

		Storage: config.StorageConfig{
			Mode:   "local",
			CdnURL: "",
		},

		Reset: config.ResetConfig{
			Exp: 2,
		},
		SMTP: testCfg.SMTP,
	}
}

func setupTestServer() {
	testConfig = loadTestConfig()
	appConfig = convertTestConfigToAppConfig(testConfig)

	testDB = config.NewDatabase(appConfig)
	testRouter = config.NewChi(appConfig)

	var err error
	testTempDir, err = os.MkdirTemp("", "chrononews_test_*")
	if err != nil {
		slog.Error("Failed to create temporary directory for tests", "err", err)
		os.Exit(1)
	}

	thumbnailsDir := filepath.Join(testTempDir, "thumbnails")
	attachmentsDir := filepath.Join(testTempDir, "attachments")
	profilesDir := filepath.Join(testTempDir, "profiles")

	_ = os.MkdirAll(thumbnailsDir, 0755)
	_ = os.MkdirAll(attachmentsDir, 0755)
	_ = os.MkdirAll(profilesDir, 0755)

	appConfig.Storage.Thumbnail = thumbnailsDir
	appConfig.Storage.Attachment = attachmentsDir
	appConfig.Storage.Profile = profilesDir

	validator := config.NewValidator()
	client := config.NewClient()

	bootstrap.Init(testRouter, testDB, appConfig, validator, client, nil)

	if err := config.Migrate(context.Background(), testDB); err != nil {
		slog.Error("Failed to migrate database for tests", "err", err)
		os.Exit(1)
	}
}

func getAuthToken(t *testing.T, db *gorm.DB, serverURL, email, role string) (string, error) {
	var user entity.User
	err := db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password!23"), bcrypt.DefaultCost)
			user = entity.User{
				Name:     fmt.Sprintf("Test %s", role),
				Email:    email,
				Password: string(hashedPassword),
				Role:     role,
			}
			if err := db.Create(&user).Error; err != nil {
				return "", fmt.Errorf("failed to create test user: %v", err)
			}
		} else {
			return "", fmt.Errorf("failed to find test user: %v", err)
		}
	}

	loginData := model.UserLogin{
		Email:        email,
		Password:     "Password!23",
		TokenCaptcha: "Token_Captcha",
	}
	body, err := json.Marshal(loginData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal login data: %v", err)
	}

	client := config.NewClient()
	resp, err := client.Post(serverURL+"/api/user/login", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to login: %v", err)
	}
	defer func() {
		err := resp.Body.Close()
		assert.NoError(t, err)
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed with status: %s", resp.Status)
	}

	var result struct {
		Data string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode login response: %v", err)
	}

	if result.Data == "" {
		return "", fmt.Errorf("token not found in login response")
	}

	return result.Data, nil
}

func TestMain(m *testing.M) {
	setupTestServer()

	exitCode := m.Run()

	if err := os.RemoveAll(testTempDir); err != nil {
		slog.Error("Failed to clean up temporary test directory", "path", testTempDir, "err", err)
	}

	os.Exit(exitCode)
}

func clearTables(db *gorm.DB) {
	db.Exec("DELETE FROM source_files_to_delete")
	db.Exec("DELETE FROM dead_letter_queue")
	db.Exec("DELETE FROM reset")
	db.Exec("DELETE FROM file")
	db.Exec("DELETE FROM post")
	db.Exec("DELETE FROM category")
	db.Exec("DELETE FROM \"user\"")
}
