package main

import (
	"chronoverseapi/internal/config"
	"log"
	"log/slog"
	"net/http"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(4)
	viper := config.NewViper()
	db := config.NewDatabase(viper)
	chi := config.NewChi(viper)
	validator := config.NewValidator()
	client := config.NewClient()
	config.Bootstrap(&config.BootstrapConfig{App: chi, DB: db, Config: viper, Validator: validator, Client: client})
	slog.Info("server run on port " + viper.GetString("web.port"))
	err := http.ListenAndServe("0.0.0.0:"+viper.GetString("web.port"), chi)
	if err != nil {
		log.Fatal(err)
	}
}
