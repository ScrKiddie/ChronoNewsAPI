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
	"net/http"
)

type BootstrapConfig struct {
	App       *chi.Mux
	DB        *gorm.DB
	Config    *viper.Viper
	Validator *validator.Validate
	Client    *http.Client
}

func Bootstrap(b *BootstrapConfig) {
	//repository
	userRepository := repository.NewUserRepository()
	categoryRepository := repository.NewCategoryRepository()
	fileRepository := repository.NewFileRepository()
	postRepository := repository.NewPostRepository()

	//adapter
	fileStorage := adapter.NewFileStorage()
	captcha := adapter.NewCaptcha(b.Client)
	//service
	userService := service.NewUserService(b.DB, userRepository, postRepository, fileStorage, captcha, b.Validator, b.Config)
	categoryService := service.NewCategoryService(b.DB, categoryRepository, userRepository, postRepository, b.Validator)
	postService := service.NewPostService(b.DB, postRepository, userRepository, fileRepository, categoryRepository, fileStorage, b.Validator, b.Config)

	//controller
	userController := controller.NewUserController(userService)
	categoryController := controller.NewCategoryController(categoryService)
	postController := controller.NewPostController(postService)

	//middleware
	userMiddleware := middleware.NewUserMiddleware(userService)

	router := route.Route{App: b.App, UserController: userController, UserMiddleware: userMiddleware, CategoryController: categoryController, PostController: postController}
	router.Setup()
}
