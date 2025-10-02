package config

import (
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Web struct {
		Port        string `mapstructure:"port"`
		CorsOrigins string `mapstructure:"cors_origins"`
	} `mapstructure:"web"`
	DB struct {
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Name     string `mapstructure:"name"`
	} `mapstructure:"db"`
	JWT struct {
		Secret string `mapstructure:"secret"`
		Exp    int    `mapstructure:"exp"`
	} `mapstructure:"jwt"`
	Captcha struct {
		Secret string `mapstructure:"secret"`
	} `mapstructure:"captcha"`
	Storage struct {
		Post    string `mapstructure:"post"`
		Profile string `mapstructure:"profile"`
	} `mapstructure:"storage"`
	Reset struct {
		Exp        int    `mapstructure:"exp"`
		URL        string `mapstructure:"url"`
		Query      string `mapstructure:"query"`
		RequestURL string `mapstructure:"request_url"`
	} `mapstructure:"reset"`
	SMTP struct {
		Host      string `mapstructure:"host"`
		Port      int    `mapstructure:"port"`
		FromName  string `mapstructure:"from_name"`
		FromEmail string `mapstructure:"from_email"`
		Username  string `mapstructure:"username"`
		Password  string `mapstructure:"password"`
	} `mapstructure:"smtp"`
}

func NewConfig() *Config {
	config := viper.New()
	config.AutomaticEnv()
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	config.SetConfigName("config")
	config.SetConfigType("json")
	config.AddConfigPath("./../")
	config.AddConfigPath("./")

	if err := config.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			slog.Info("Config file not found; using environment variables")
		} else {
			slog.Error("Error reading config file", "err", err)
			os.Exit(1)
		}
	}

	var appConfig Config
	if err := config.Unmarshal(&appConfig); err != nil {
		slog.Error("Error unmarshalling config", "err", err)
		os.Exit(1)
	}

	return &appConfig
}
