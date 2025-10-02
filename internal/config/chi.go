package config

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	slogchi "github.com/samber/slog-chi"
)

func NewChi(config *Config) *chi.Mux {
	r := chi.NewRouter()
	r.Use(slogchi.New(slog.Default()))

	originsStr := config.Web.CorsOrigins
	var origins []string
	if originsStr != "" {
		origins = strings.Split(originsStr, ",")
	}

	allowedOrigins := origins
	for _, origin := range origins {
		if origin == "*" {
			allowedOrigins = []string{"*"}
			break
		}
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		err := json.NewEncoder(w).Encode(map[string]string{
			"error": "Not found",
		})
		if err != nil {
			slog.Error("Failed to encode not found response", "err", err)
		}
	})
	r.Use(middleware.Recoverer)

	return r
}
