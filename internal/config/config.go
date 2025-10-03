package config

import (
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type FromConfig struct {
	Name  string `mapstructure:"name"`
	Email string `mapstructure:"email"`
}

type WebConfig struct {
	Port        string `mapstructure:"port"`
	CorsOrigins string `mapstructure:"cors_origins"`
}

type DBConfig struct {
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Name     string `mapstructure:"name"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Exp    int    `mapstructure:"exp"`
}

type CaptchaConfig struct {
	Secret string `mapstructure:"secret"`
}

type StorageConfig struct {
	Post    string `mapstructure:"post"`
	Profile string `mapstructure:"profile"`
}

type ResetConfig struct {
	Exp        int    `mapstructure:"exp"`
	URL        string `mapstructure:"url"`
	Query      string `mapstructure:"query"`
	RequestURL string `mapstructure:"request_url"`
}

type SMTPConfig struct {
	Host     string     `mapstructure:"host"`
	Port     int        `mapstructure:"port"`
	From     FromConfig `mapstructure:"from"`
	Username string     `mapstructure:"username"`
	Password string     `mapstructure:"password"`
}

type Config struct {
	Web     WebConfig     `mapstructure:"web"`
	DB      DBConfig      `mapstructure:"db"`
	JWT     JWTConfig     `mapstructure:"jwt"`
	Captcha CaptchaConfig `mapstructure:"captcha"`
	Storage StorageConfig `mapstructure:"storage"`
	Reset   ResetConfig   `mapstructure:"reset"`
	SMTP    SMTPConfig    `mapstructure:"smtp"`
}

func NewConfig() *Config {
	config := viper.New()
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	envKeys := []string{
		"web.port", "web.cors_origins",
		"db.user", "db.password", "db.host", "db.port", "db.name",
		"jwt.secret", "jwt.exp",
		"captcha.secret",
		"storage.post", "storage.profile",
		"reset.exp", "reset.url", "reset.query", "reset.request_url",
		"smtp.host", "smtp.port", "smtp.username", "smtp.password",
		"smtp.from.name", "smtp.from.email",
	}

	for _, key := range envKeys {
		if err := config.BindEnv(key); err != nil {
			slog.Error("Failed to bind environment variable", "key", key, "error", err)
			os.Exit(1)
		}
	}

	config.SetConfigName("config")
	config.SetConfigType("json")
	config.AddConfigPath("./../")
	config.AddConfigPath("./")

	if err := config.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			slog.Info("Config file not found; using environment variables")
		}
	}

	var appConfig Config
	if err := config.Unmarshal(&appConfig); err != nil {
		slog.Error("Error unmarshalling config", "err", err)
		os.Exit(1)
	}

	if err := validateConfig(&appConfig); err != nil {
		slog.Error("Configuration validation failed", "error", err)
		os.Exit(1)
	}

	return &appConfig
}

func validateConfig(cfg *Config) error {
	var missingFields []string

	if cfg.Web.Port == "" {
		missingFields = append(missingFields, "web.port")
	}

	if cfg.DB.Host == "" {
		missingFields = append(missingFields, "db.host")
	}
	if cfg.DB.Port <= 0 {
		missingFields = append(missingFields, "db.port")
	}
	if cfg.DB.User == "" {
		missingFields = append(missingFields, "db.user")
	}
	if cfg.DB.Name == "" {
		missingFields = append(missingFields, "db.name")
	}

	if cfg.JWT.Secret == "" {
		missingFields = append(missingFields, "jwt.secret")
	}
	if cfg.JWT.Exp <= 0 {
		missingFields = append(missingFields, "jwt.exp")
	}

	if cfg.Captcha.Secret == "" {
		missingFields = append(missingFields, "captcha.secret")
	}

	if cfg.Reset.URL == "" {
		missingFields = append(missingFields, "reset.url")
	}
	if cfg.Reset.Query == "" {
		missingFields = append(missingFields, "reset.query")
	}
	if cfg.Reset.RequestURL == "" {
		missingFields = append(missingFields, "reset.request_url")
	}
	if cfg.Reset.Exp <= 0 {
		missingFields = append(missingFields, "reset.exp")
	}

	if cfg.SMTP.Host == "" {
		missingFields = append(missingFields, "smtp.host")
	}
	if cfg.SMTP.Port <= 0 {
		missingFields = append(missingFields, "smtp.port")
	}
	if cfg.SMTP.From.Name == "" {
		missingFields = append(missingFields, "smtp.from.name")
	}
	if cfg.SMTP.From.Email == "" {
		missingFields = append(missingFields, "smtp.from.email")
	}

	if len(missingFields) > 0 {
		return errors.New("missing required configuration fields: " + strings.Join(missingFields, ", "))
	}

	return nil
}
