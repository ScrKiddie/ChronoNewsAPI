package main

import (
	"chrononewsapi/internal/bootstrap"
	"chrononewsapi/internal/config"
	"log"
	"log/slog"
	"net/http"
)

func main() {
	appConfig := config.NewConfig()
	db := config.NewDatabase(appConfig)
	chi := config.NewChi(appConfig)
	validator := config.NewValidator()
	client := config.NewClient()
	bootstrap.Init(chi, db, appConfig, validator, client)
	slog.Info("Server run on port " + appConfig.Web.Port)
	err := http.ListenAndServe("0.0.0.0:"+appConfig.Web.Port, chi)
	if err != nil {
		log.Fatal(err)
	}
}
