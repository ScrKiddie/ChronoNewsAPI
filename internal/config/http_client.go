package config

import (
	"net/http"
	"time"
)

func NewClient() *http.Client {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	return client
}
