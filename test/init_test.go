package test

import (
	"bytes"
	"chrononewsapi/internal/bootstrap"
	"chrononewsapi/internal/config"
	"chrononewsapi/internal/entity"
	"chrononewsapi/internal/model"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func NewTestConfig() *config.Config {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.SetConfigName("config")
	v.SetConfigType("json")
	v.AddConfigPath("../")
	v.AddConfigPath("./")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			slog.Error("Config file not found; ensure config.json exists in the root directory")
			os.Exit(1)
		} else {
			slog.Error("Error reading config file", "err", err)
			os.Exit(1)
		}
	}

	testConfig := v.Sub("test")
	if testConfig == nil {
		slog.Error("'test' configuration not found in config.json")
		os.Exit(1)
	}

	var appConfig config.Config
	if err := testConfig.Unmarshal(&appConfig); err != nil {
		slog.Error("Error unmarshalling test config", "err", err)
		os.Exit(1)
	}

	return &appConfig
}

func TestNewTestConfig(t *testing.T) {
	cfg := NewTestConfig()

	if cfg == nil {
		t.Fatal("NewTestConfig returned nil")
	}

	expectedCaptchaSecret := "1x0000000000000000000000000000000AA" // this value is from https://developers.cloudflare.com/turnstile/troubleshooting/testing/
	if cfg.Captcha.Secret != expectedCaptchaSecret {
		t.Errorf("expected captcha secret to be '%s', but got '%s'", expectedCaptchaSecret, cfg.Captcha.Secret)
	}

	expectedDbName := "chronoverse_test"
	if cfg.DB.Name != expectedDbName {
		t.Errorf("expected db name to be '%s', but got '%s'", expectedDbName, cfg.DB.Name)
	}
}

var (
	testDB     *gorm.DB
	testRouter *chi.Mux
	testConfig *config.Config
)

func setupTestServer() {
	testConfig = NewTestConfig()
	testDB = config.NewDatabase(testConfig)
	testRouter = config.NewChi(testConfig)
	validator := config.NewValidator()
	client := config.NewClient()
	bootstrap.Init(testRouter, testDB, testConfig, validator, client)
	testDB.AutoMigrate(&entity.User{}, &entity.Category{}, &entity.Post{}, &entity.File{}, &entity.Reset{})
}

func getAuthToken(db *gorm.DB, serverURL, email, role string) (string, error) {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password!23"), bcrypt.DefaultCost)
	user := entity.User{
		Name:     fmt.Sprintf("Test %s", role),
		Email:    email,
		Password: string(hashedPassword),
		Role:     role,
	}
	if err := db.Create(&user).Error; err != nil {
		return "", fmt.Errorf("failed to create test user: %v", err)
	}

	loginData := model.UserLogin{
		Email:        email,
		Password:     "Password!23",
		TokenCaptcha: "1x0000000000000000000000000000000AA", // this value is from https://developers.cloudflare.com/turnstile/troubleshooting/testing/
	}
	body, _ := json.Marshal(loginData)

	resp, err := http.Post(serverURL+"/api/user/login", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to login: %v", err)
	}
	defer resp.Body.Close()

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

	if testConfig.Storage.Post != "" {
		cleanedPath := filepath.Clean(testConfig.Storage.Post)
		storageRoot := strings.Split(cleanedPath, string(os.PathSeparator))[0]
		if storageRoot != "" && storageRoot != "." && storageRoot != "/" {
			if err := os.RemoveAll(storageRoot); err != nil {
				slog.Error("Failed to clean up dynamic storage directory", "path", storageRoot, "err", err)
			}
		}
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
