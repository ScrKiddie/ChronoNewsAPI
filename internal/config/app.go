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
	resetRepository := repository.NewResetRepository()

	//adapter
	fileStorage := adapter.NewStorageAdapter()
	captcha := adapter.NewCaptchaAdapter(b.Client)
	emailAdapter := adapter.NewEmailAdapter()

	//service
	userService := service.NewUserService(b.DB, userRepository, postRepository, fileStorage, captcha, b.Validator, b.Config)
	categoryService := service.NewCategoryService(b.DB, categoryRepository, userRepository, postRepository, b.Validator)
	postService := service.NewPostService(b.DB, postRepository, userRepository, fileRepository, categoryRepository, fileStorage, b.Validator, b.Config)
	resetService := service.NewResetService(b.DB, resetRepository, userRepository, emailAdapter, b.Validator, b.Config)

	//controller
	userController := controller.NewUserController(userService)
	categoryController := controller.NewCategoryController(categoryService)
	postController := controller.NewPostController(postService)
	resetController := controller.NewResetController(resetService)

	//middleware
	userMiddleware := middleware.NewUserMiddleware(userService)

	router := route.Route{App: b.App, UserController: userController, UserMiddleware: userMiddleware, CategoryController: categoryController, PostController: postController, ResetController: resetController}
	router.Setup()
}
