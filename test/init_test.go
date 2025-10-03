package test

import (
	"bytes"
	"chrononewsapi/internal/bootstrap"
	"chrononewsapi/internal/config"
	"chrononewsapi/internal/entity"
	"chrononewsapi/internal/model"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type CaptchaSecretConfig struct {
	Pass  string `mapstructure:"pass"`
	Fail  string `mapstructure:"fail"`
	Usage string `mapstructure:"usage"`
}

type TestConfig struct {
	Secret CaptchaSecretConfig `mapstructure:"secret"`
}

func loadTestConfig() (*config.Config, *TestConfig) {
	v := viper.New()

	v.SetEnvPrefix("TEST")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.SetConfigName("config")
	v.SetConfigType("json")
	v.AddConfigPath("../")

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			slog.Error("Failed to read config file for test", "error", err)
			os.Exit(1)
		}
	}

	testSettings := v.Sub("test").AllSettings()
	if len(testSettings) == 0 {
		slog.Error("Configuration 'test' block not found or is empty. Ensure it exists in config.json or is set via TEST_* env vars.")
		os.Exit(1)
	}

	var captchaCfg TestConfig
	if err := v.Sub("test").Sub("captcha").Unmarshal(&captchaCfg); err != nil {
		slog.Error("Failed to unmarshal custom test config", "error", err)
		os.Exit(1)
	}

	delete(testSettings, "captcha")

	var appCfg config.Config
	vTemp := viper.New()
	if err := vTemp.MergeConfigMap(testSettings); err != nil {
		slog.Error("Failed to merge test config map", "error", err)
		os.Exit(1)
	}
	if err := vTemp.Unmarshal(&appCfg); err != nil {
		slog.Error("Failed to unmarshal app config for test", "error", err)
		os.Exit(1)
	}
	return &appCfg, &captchaCfg
}

var (
	testDB            *gorm.DB
	testRouter        *chi.Mux
	testCaptchaConfig *TestConfig
	appConfig         *config.Config
	testTempDir       string
)

func setupTestServer() {
	appConfig, testCaptchaConfig = loadTestConfig()

	appConfig.Captcha.Secret = testCaptchaConfig.Secret.Pass

	testDB = config.NewDatabase(appConfig)
	testRouter = config.NewChi(appConfig)

	var err error
	testTempDir, err = os.MkdirTemp("", "chrononews_test_*")
	if err != nil {
		slog.Error("Failed to create temporary directory for tests", "err", err)
		os.Exit(1)
	}

	appConfig.Storage.Post = filepath.Join(testTempDir, "posts")
	appConfig.Storage.Profile = filepath.Join(testTempDir, "profiles")

	validator := config.NewValidator()
	client := config.NewClient()
	bootstrap.Init(testRouter, testDB, appConfig, validator, client)
	err = testDB.AutoMigrate(&entity.User{}, &entity.Category{}, &entity.Post{}, &entity.File{}, &entity.Reset{})
	if err != nil {
		slog.Error("Failed to auto migrate database for tests", "err", err)
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
	db.Where("1 = 1").Delete(&entity.Reset{})
	db.Where("1 = 1").Delete(&entity.File{})
	db.Where("1 = 1").Delete(&entity.Post{})
	db.Where("1 = 1").Delete(&entity.Category{})
	db.Where("1 = 1").Delete(&entity.User{})
}
