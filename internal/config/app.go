package config

import (
	"chronoverseapi/internal/adapter"
	"chronoverseapi/internal/controller"
	"chronoverseapi/internal/middleware"
	"chronoverseapi/internal/repository"
	"chronoverseapi/internal/route"
	"chronoverseapi/internal/service"
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
	categoryRepository := repository.NewCategoryRepository()

	//adapter
	fileStorage := adapter.NewFileStorage()

	//service
	userService := service.NewUserService(b.DB, userRepository, fileStorage, b.Validator, b.Config)
	categoryService := service.NewCategoryService(b.DB, categoryRepository, userRepository, b.Validator)

	//controller
	userController := controller.NewUserController(userService)
	categoryController := controller.NewCategoryController(categoryService)

	//middleware
	userMiddleware := middleware.NewUserMiddleware(userService)

	router := route.Route{App: b.App, UserController: userController, UserMiddleware: userMiddleware, CategoryController: categoryController}
	router.Setup()
}
