package main

import (
	"chrononewsapi/internal/config"
	"log"
	"log/slog"
	"net/http"
)

func main() {
	viper := config.NewViper()
	db := config.NewDatabase(viper)
	chi := config.NewChi(viper)
	validator := config.NewValidator()
	client := config.NewClient()
	config.Bootstrap(&config.BootstrapConfig{App: chi, DB: db, Config: viper, Validator: validator, Client: client})
	slog.Info("Server run on port " + viper.GetString("web.port"))
	err := http.ListenAndServe("0.0.0.0:"+viper.GetString("web.port"), chi)
	if err != nil {
		log.Fatal(err)
	}
}
