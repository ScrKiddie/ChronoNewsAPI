package main

import (
	"chrononewsapi/internal/bootstrap"
	"chrononewsapi/internal/config"
	"log"
	"log/slog"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	appConfig := config.NewConfig()
	db := config.NewDatabase(appConfig)
	chi := config.NewChi(appConfig)
	validator := config.NewValidator()
	httpClient := config.NewClient()

	var s3Client *s3.Client
	if appConfig.Storage.Mode == "s3" {
		var err error
		s3Client, err = config.NewS3Client(appConfig.Storage.S3)
		if err != nil {
			log.Fatalf("failed to create s3 client: %v", err)
		}
	}

	bootstrap.Init(chi, db, appConfig, validator, httpClient, s3Client)
	slog.Info("Server run on port " + appConfig.Web.Port)
	err := http.ListenAndServe("0.0.0.0:"+appConfig.Web.Port, chi)
	if err != nil {
		log.Fatal(err)
	}
}
