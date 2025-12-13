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

type ClientPathConfig struct {
	Post     string `mapstructure:"post"`
	Category string `mapstructure:"category"`
	Reset    string `mapstructure:"reset"`
	Forgot   string `mapstructure:"forgot"`
}

type WebConfig struct {
	BaseURL     string           `mapstructure:"base_url"`
	Port        string           `mapstructure:"port"`
	CorsOrigins string           `mapstructure:"cors_origins"`
	ClientURL   string           `mapstructure:"client_url"`
	ClientPaths ClientPathConfig `mapstructure:"client_paths"`
}

type DBConfig struct {
	User      string `mapstructure:"user"`
	Password  string `mapstructure:"password"`
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Name      string `mapstructure:"name"`
	SslMode   string `mapstructure:"sslmode"`
	Migration bool   `mapstructure:"migration"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Exp    int    `mapstructure:"exp"`
}

type CaptchaConfig struct {
	Secret string `mapstructure:"secret"`
}

type S3Config struct {
	Bucket    string `mapstructure:"bucket"`
	Region    string `mapstructure:"region"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Endpoint  string `mapstructure:"endpoint"`
}

type StorageConfig struct {
	Mode    string   `mapstructure:"mode"`
	CdnURL  string   `mapstructure:"cdn_url"`
	Post    string   `mapstructure:"post"`
	Profile string   `mapstructure:"profile"`
	S3      S3Config `mapstructure:"s3"`
}

type ResetConfig struct {
	Exp int `mapstructure:"exp"`
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
		"web.base_url", "web.port", "web.cors_origins", "web.client_url",
		"web.client_paths.post", "web.client_paths.category", "web.client_paths.reset", "web.client_paths.forgot",

		"db.user", "db.password", "db.host", "db.port", "db.name", "db.sslmode", "db.migration",

		"jwt.secret", "jwt.exp",

		"captcha.secret",

		"storage.mode", "storage.cdn_url", "storage.post", "storage.profile",
		"storage.s3.bucket", "storage.s3.region", "storage.s3.access_key", "storage.s3.secret_key", "storage.s3.endpoint",

		"reset.exp",

		"smtp.host", "smtp.port", "smtp.username", "smtp.password",
		"smtp.from.name", "smtp.from.email",
	}

	for _, key := range envKeys {
		if err := config.BindEnv(key); err != nil {
			slog.Error("Failed to bind environment variable", "key", key, "error", err)
			os.Exit(1)
		}
	}

	config.SetDefault("storage.mode", "local")
	config.SetDefault("db.sslmode", "require")
	config.SetDefault("db.migration", false)

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

	if cfg.Web.BaseURL == "" {
		missingFields = append(missingFields, "web.base_url")
	}
	if cfg.Web.Port == "" {
		missingFields = append(missingFields, "web.port")
	}
	if cfg.Web.ClientURL == "" {
		missingFields = append(missingFields, "web.client_url")
	}
	if cfg.Web.ClientPaths.Post == "" {
		missingFields = append(missingFields, "web.client_paths.post")
	}
	if cfg.Web.ClientPaths.Category == "" {
		missingFields = append(missingFields, "web.client_paths.category")
	}
	if cfg.Web.ClientPaths.Reset == "" {
		missingFields = append(missingFields, "web.client_paths.reset")
	}
	if cfg.Web.ClientPaths.Forgot == "" {
		missingFields = append(missingFields, "web.client_paths.forgot")
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
	if cfg.DB.SslMode == "" {
		missingFields = append(missingFields, "db.sslmode")
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

	if cfg.Storage.Mode == "s3" {
		if cfg.Storage.S3.Bucket == "" {
			missingFields = append(missingFields, "storage.s3.bucket")
		}
		if cfg.Storage.S3.Region == "" {
			slog.Warn("storage.s3.region is not set, defaulting to 'auto' for R2 compatibility")
			cfg.Storage.S3.Region = "auto"
		}
		if cfg.Storage.S3.AccessKey == "" {
			missingFields = append(missingFields, "storage.s3.access_key")
		}
		if cfg.Storage.S3.SecretKey == "" {
			missingFields = append(missingFields, "storage.s3.secret_key")
		}
		if cfg.Storage.S3.Endpoint == "" {
			missingFields = append(missingFields, "storage.s3.endpoint (required for Cloudflare R2)")
		}

		if cfg.Storage.CdnURL == "" {
			missingFields = append(missingFields, "storage.cdn_url (required for S3 mode)")
		}
	} else if cfg.Storage.Mode == "local" {
		if cfg.Storage.Post == "" {
			missingFields = append(missingFields, "storage.post")
		}
		if cfg.Storage.Profile == "" {
			missingFields = append(missingFields, "storage.profile")
		}
	} else if cfg.Storage.Mode != "" {
		missingFields = append(missingFields, "storage.mode (must be 'local' or 's3')")
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
