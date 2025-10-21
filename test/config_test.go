package test

import (
	"chrononewsapi/internal/config"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type TestCaptchaSecretConfig struct {
	Pass  string `mapstructure:"pass"`
	Fail  string `mapstructure:"fail"`
	Usage string `mapstructure:"usage"`
}

type TestCaptchaConfig struct {
	Secret TestCaptchaSecretConfig `mapstructure:"secret"`
}

type TestConfig struct {
	JWT     config.JWTConfig     `mapstructure:"jwt"`
	Web     config.WebConfig     `mapstructure:"web"`
	Captcha TestCaptchaConfig    `mapstructure:"captcha"`
	DB      config.DBConfig      `mapstructure:"db"`
	Reset   config.ResetConfig   `mapstructure:"reset"`
	SMTP    config.SMTPConfig    `mapstructure:"smtp"`
	Storage config.StorageConfig `mapstructure:"storage"`
}

type TestConfigWrapper struct {
	Test TestConfig `mapstructure:"test"`
}

func loadTestConfig() *TestConfig {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	envKeys := []string{
		"test.jwt.secret", "test.jwt.exp",
		"test.web.port", "test.web.cors_origins",
		"test.captcha.secret.pass", "test.captcha.secret.fail", "test.captcha.secret.usage",
		"test.db.user", "test.db.password", "test.db.host", "test.db.port", "test.db.name",
		"test.reset.url", "test.reset.query", "test.reset.request_url", "test.reset.exp",
		"test.smtp.host", "test.smtp.port", "test.smtp.username", "test.smtp.password",
		"test.smtp.from.name", "test.smtp.from.email",
		"test.storage.profile", "test.storage.post",
	}

	for _, key := range envKeys {
		if err := v.BindEnv(key); err != nil {
			slog.Error("Failed to bind environment variable", "key", key, "error", err)
			os.Exit(1)
		}
	}

	v.SetConfigName("config")
	v.SetConfigType("json")
	v.AddConfigPath("../")

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			slog.Info("Test config file not found; using environment variables")
		} else {
			slog.Warn("Failed to read test config file, will proceed with env vars", "error", err)
		}
	}

	var testCfgWrapper TestConfigWrapper
	if err := v.Unmarshal(&testCfgWrapper); err != nil {
		slog.Error("Failed to unmarshal test config. Check TEST_* env vars or 'test' block in config.json.", "error", err)
		os.Exit(1)
	}

	if err := validateTestConfig(&testCfgWrapper.Test); err != nil {
		slog.Error("Test configuration validation failed", "error", err)
		os.Exit(1)
	}

	return &testCfgWrapper.Test
}

func validateTestConfig(cfg *TestConfig) error {
	var missingFields []string

	if cfg.JWT.Secret == "" {
		missingFields = append(missingFields, "test.jwt.secret")
	}
	if cfg.JWT.Exp <= 0 {
		missingFields = append(missingFields, "test.jwt.exp")
	}

	if cfg.Web.Port == "" {
		missingFields = append(missingFields, "test.web.port")
	}

	if cfg.DB.Host == "" {
		missingFields = append(missingFields, "test.db.host")
	}
	if cfg.DB.Port <= 0 {
		missingFields = append(missingFields, "test.db.port")
	}
	if cfg.DB.User == "" {
		missingFields = append(missingFields, "test.db.user")
	}
	if cfg.DB.Name == "" {
		missingFields = append(missingFields, "test.db.name")
	}

	if cfg.Captcha.Secret.Pass == "" {
		missingFields = append(missingFields, "test.captcha.secret.pass")
	}
	if cfg.Captcha.Secret.Fail == "" {
		missingFields = append(missingFields, "test.captcha.secret.fail")
	}
	if cfg.Captcha.Secret.Usage == "" {
		missingFields = append(missingFields, "test.captcha.secret.usage")
	}

	if cfg.Reset.URL == "" {
		missingFields = append(missingFields, "test.reset.url")
	}
	if cfg.Reset.Query == "" {
		missingFields = append(missingFields, "test.reset.query")
	}
	if cfg.Reset.RequestURL == "" {
		missingFields = append(missingFields, "test.reset.request_url")
	}
	if cfg.Reset.Exp <= 0 {
		missingFields = append(missingFields, "test.reset.exp")
	}

	if cfg.SMTP.Host == "" {
		missingFields = append(missingFields, "test.smtp.host")
	}
	if cfg.SMTP.Port <= 0 {
		missingFields = append(missingFields, "test.smtp.port")
	}
	if cfg.SMTP.From.Name == "" {
		missingFields = append(missingFields, "test.smtp.from.name")
	}
	if cfg.SMTP.From.Email == "" {
		missingFields = append(missingFields, "test.smtp.from.email")
	}

	if len(missingFields) > 0 {
		return fmt.Errorf("missing required configuration fields: %s", strings.Join(missingFields, ", "))
	}

	return nil
}
