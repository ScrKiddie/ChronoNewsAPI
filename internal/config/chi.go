package config

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	slogchi "github.com/samber/slog-chi"
	"github.com/spf13/viper"
)

func NewChi(config *viper.Viper) *chi.Mux {
	r := chi.NewRouter()
	r.Use(slogchi.New(slog.Default()))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   config.GetStringSlice("web.cors.origins"),
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
			slog.Error("failed to encode not found response", "err", err)
		}
	})
	r.Use(middleware.Recoverer)

	return r
}
