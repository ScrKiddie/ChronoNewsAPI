package config

import (
	"ChronoverseAPI/internal/adapter"
	"ChronoverseAPI/internal/controller"
	"ChronoverseAPI/internal/middleware"
	"ChronoverseAPI/internal/repository"
	"ChronoverseAPI/internal/route"
	"ChronoverseAPI/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type BootstrapConfig struct {
	App       *chi.Mux
	DB        *gorm.DB
	Config    *viper.Viper
	Validator *validator.Validate
}

func Bootstrap(b *BootstrapConfig) {
	//repository
	userRepository := repository.NewUserRepository()

	//adapter
	fileStorage := adapter.NewFileStorage()

	//service
	userService := service.NewUserService(b.DB, userRepository, fileStorage, b.Validator, b.Config)

	//controller
	userController := controller.NewUserController(userService)

	//middleware
	userMiddleware := middleware.NewUserMiddleware(userService)

	router := route.Route{App: b.App, UserController: userController, UserMiddleware: userMiddleware}
	router.Setup()
}
