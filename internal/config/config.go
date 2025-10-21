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
	Post       string `mapstructure:"post"`
	Profile    string `mapstructure:"profile"`
	Upload     string `mapstructure:"upload"`
	Compressed string `mapstructure:"compressed"`
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

type CompressionConfig struct {
	IsConcurrent bool   `mapstructure:"is_concurrent"`
	NumWorkers   int    `mapstructure:"num_workers"`
	MaxWidth     int    `mapstructure:"max_width"`
	MaxHeight    int    `mapstructure:"max_height"`
	WebPQuality  int    `mapstructure:"webp_quality"`
	MaxRetries   int    `mapstructure:"max_retries"`
	LogLevel     string `mapstructure:"log_level"`
}

type RabbitMQConfig struct {
	URL          string `mapstructure:"url"`
	Enabled      bool   `mapstructure:"enabled"`
	QueueName    string `mapstructure:"queue_name"`
	BatchSize    int    `mapstructure:"batch_size"`
	BatchTimeout int    `mapstructure:"batch_timeout"`
}

type Config struct {
	Web         WebConfig         `mapstructure:"web"`
	DB          DBConfig          `mapstructure:"db"`
	JWT         JWTConfig         `mapstructure:"jwt"`
	Captcha     CaptchaConfig     `mapstructure:"captcha"`
	Storage     StorageConfig     `mapstructure:"storage"`
	Reset       ResetConfig       `mapstructure:"reset"`
	SMTP        SMTPConfig        `mapstructure:"smtp"`
	Compression CompressionConfig `mapstructure:"compression"`
	RabbitMQ    RabbitMQConfig    `mapstructure:"rabbitmq"`
}

func NewConfig() *Config {
	config := viper.New()
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	envKeys := []string{
		"web.port", "web.cors_origins",
		"db.user", "db.password", "db.host", "db.port", "db.name",
		"jwt.secret", "jwt.exp",
		"captcha.secret",
		"storage.post", "storage.profile", "storage.upload", "storage.compressed",
		"reset.exp", "reset.url", "reset.query", "reset.request_url",
		"smtp.host", "smtp.port", "smtp.username", "smtp.password",
		"smtp.from.name", "smtp.from.email",
		"compression.is_concurrent", "compression.num_workers",
		"compression.max_width", "compression.max_height",
		"compression.webp_quality", "compression.max_retries",
		"compression.log_level",
		"rabbitmq.url", "rabbitmq.enabled", "rabbitmq.queue_name",
		"rabbitmq.batch_size", "rabbitmq.batch_timeout",
	}

	for _, key := range envKeys {
		if err := config.BindEnv(key); err != nil {
			slog.Error("bind env error", "key", key, "error", err)
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
			slog.Info("config file not found using env")
		}
	}

	var appConfig Config
	if err := config.Unmarshal(&appConfig); err != nil {
		slog.Error("config unmarshal error", "error", err)
		os.Exit(1)
	}

	if appConfig.Compression.MaxWidth == 0 {
		appConfig.Compression.MaxWidth = 1920
	}
	if appConfig.Compression.MaxHeight == 0 {
		appConfig.Compression.MaxHeight = 1080
	}
	if appConfig.Compression.WebPQuality == 0 {
		appConfig.Compression.WebPQuality = 80
	}
	if appConfig.Compression.MaxRetries == 0 {
		appConfig.Compression.MaxRetries = 3
	}
	if appConfig.Compression.NumWorkers == 0 {
		appConfig.Compression.NumWorkers = 4
	}
	if appConfig.Compression.LogLevel == "" {
		appConfig.Compression.LogLevel = "info"
	}

	if appConfig.RabbitMQ.QueueName == "" {
		appConfig.RabbitMQ.QueueName = "image_compression_queue"
	}
	if appConfig.RabbitMQ.BatchSize == 0 {
		appConfig.RabbitMQ.BatchSize = 50
	}
	if appConfig.RabbitMQ.BatchTimeout == 0 {
		appConfig.RabbitMQ.BatchTimeout = 30
	}

	if err := validateConfig(&appConfig); err != nil {
		slog.Error("config validation failed", "error", err)
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

	if cfg.Compression.MaxWidth <= 0 {
		missingFields = append(missingFields, "compression.max_width")
	}
	if cfg.Compression.MaxHeight <= 0 {
		missingFields = append(missingFields, "compression.max_height")
	}
	if cfg.Compression.WebPQuality <= 0 || cfg.Compression.WebPQuality > 100 {
		missingFields = append(missingFields, "compression.webp_quality (must be 1-100)")
	}
	if cfg.Compression.MaxRetries <= 0 {
		missingFields = append(missingFields, "compression.max_retries (must be > 0)")
	}
	if cfg.Compression.NumWorkers <= 0 {
		missingFields = append(missingFields, "compression.num_workers")
	}

	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[cfg.Compression.LogLevel] {
		missingFields = append(missingFields, "compression.log_level (must be: debug, info, warn, or error)")
	}

	if cfg.RabbitMQ.Enabled {
		if cfg.RabbitMQ.URL == "" {
			missingFields = append(missingFields, "rabbitmq.url (required when enabled)")
		}
		if cfg.RabbitMQ.QueueName == "" {
			missingFields = append(missingFields, "rabbitmq.queue_name")
		}
		if cfg.RabbitMQ.BatchSize <= 0 {
			missingFields = append(missingFields, "rabbitmq.batch_size (must be > 0)")
		}
		if cfg.RabbitMQ.BatchTimeout <= 0 {
			missingFields = append(missingFields, "rabbitmq.batch_timeout (must be > 0)")
		}
	}

	if len(missingFields) > 0 {
		return errors.New("missing or invalid required configuration fields: " + strings.Join(missingFields, ", "))
	}

	return nil
}
