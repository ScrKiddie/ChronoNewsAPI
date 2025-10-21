package adapter

import (
	"bytes"
	"chrononewsapi/internal/model"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"time"
)

type CaptchaAdapter struct {
	Client *http.Client
}

func NewCaptchaAdapter(client *http.Client) *CaptchaAdapter {
	return &CaptchaAdapter{Client: client}
}

func (r *CaptchaAdapter) Verify(request *model.CaptchaRequest) (bool, error) {
	const maxRetries = 3
	const initialBackoff = 1 * time.Second

	body, err := json.Marshal(request)
	if err != nil {
		return false, fmt.Errorf("failed to marshal captcha request: %w", err)
	}

	var resp *http.Response
	for attempt := 0; attempt < maxRetries; attempt++ {
		reqBody := bytes.NewBuffer(body)
		resp, err = r.Client.Post(
			"https://challenges.cloudflare.com/turnstile/v0/siteverify",
			"application/json",
			reqBody,
		)

		if err == nil {
			break
		}

		slog.Warn("Captcha verification attempt failed, retrying...", "attempt", attempt+1, "error", err)

		if attempt == maxRetries-1 {
			return false, fmt.Errorf("failed to verify captcha after %d attempts: %w", maxRetries, err)
		}

		backoff := initialBackoff * time.Duration(math.Pow(2, float64(attempt)))
		time.Sleep(backoff)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			slog.Error("Failed to close response body from captcha verification", "err", err)
		}
	}(resp.Body)

	var response model.CaptchaResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return false, fmt.Errorf("failed to decode captcha response: %w", err)
	}

	if !response.Success {
		slog.Info("Captcha verification failed: " + string(body))
	}
	return response.Success, nil
}
