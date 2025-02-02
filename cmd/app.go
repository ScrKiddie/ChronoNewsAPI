package main

import (
	"chronoverseapi/internal/config"
	"log"
	"log/slog"
	"net/http"
)

func main() {
	viper := config.NewViper()
	db := config.NewDatabase(viper)
	chi := config.NewChi()
	validator := config.NewValidator()
	config.Bootstrap(&config.BootstrapConfig{App: chi, DB: db, Config: viper, Validator: validator})
	slog.Info("server run on port " + viper.GetString("web.port"))
	err := http.ListenAndServe("0.0.0.0:"+viper.GetString("web.port"), chi)
	if err != nil {
		log.Fatal(err)
	}
}
