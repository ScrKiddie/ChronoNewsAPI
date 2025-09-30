package adapter

import (
	"bytes"
	"chrononewsapi/internal/model"
	"encoding/json"
	"log/slog"
	"net/http"
)

type CaptchaAdapter struct {
	Client *http.Client
}

func NewCaptchaAdapter(client *http.Client) *CaptchaAdapter {
	return &CaptchaAdapter{Client: client}
}

func (r *CaptchaAdapter) Verify(request *model.CaptchaRequest) (bool, error) {

	body, err := json.Marshal(request)
	if err != nil {
		return false, err
	}

	resp, err := r.Client.Post(
		"https://challenges.cloudflare.com/turnstile/v0/siteverify",
		"application/json",
		bytes.NewBuffer(body),
	)

	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var response model.CaptchaResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return false, err
	}

	if !response.Success {
		slog.Info("Captcha verification failed: " + string(body))
	}
	return response.Success, nil
}
