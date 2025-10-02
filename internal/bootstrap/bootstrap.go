package bootstrap

import (
	"chrononewsapi/internal/adapter"
	"chrononewsapi/internal/config"
	"chrononewsapi/internal/controller"
	"chrononewsapi/internal/middleware"
	"chrononewsapi/internal/repository"
	"chrononewsapi/internal/route"
	"chrononewsapi/internal/service"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

func Init(app *chi.Mux, db *gorm.DB, config *config.Config, validator *validator.Validate, client *http.Client) {
	//repository
	userRepository := repository.NewUserRepository()
	categoryRepository := repository.NewCategoryRepository()
	fileRepository := repository.NewFileRepository()
	postRepository := repository.NewPostRepository()
	resetRepository := repository.NewResetRepository()

	//adapter
	storageAdapter := adapter.NewStorageAdapter()
	captchaAdapter := adapter.NewCaptchaAdapter(client)
	emailAdapter := adapter.NewEmailAdapter()

	//service
	userService := service.NewUserService(db, userRepository, postRepository, resetRepository, storageAdapter, captchaAdapter, emailAdapter, validator, config)
	categoryService := service.NewCategoryService(db, categoryRepository, userRepository, postRepository, validator)
	postService := service.NewPostService(db, postRepository, userRepository, fileRepository, categoryRepository, storageAdapter, validator, config)
	resetService := service.NewResetService(db, resetRepository, userRepository, emailAdapter, captchaAdapter, validator, config)
	fileService := service.NewFileService(db, fileRepository, storageAdapter, config, validator)

	//controller
	userController := controller.NewUserController(userService)
	categoryController := controller.NewCategoryController(categoryService)
	postController := controller.NewPostController(postService)
	resetController := controller.NewResetController(resetService)
	fileController := controller.NewFileController(fileService)

	//middleware
	userMiddleware := middleware.NewUserMiddleware(userService)

	router := route.Route{
		App:                app,
		UserController:     userController,
		UserMiddleware:     userMiddleware,
		CategoryController: categoryController,
		PostController:     postController,
		ResetController:    resetController,
		FileController:     fileController,
		Config:             config,
	}
	router.Setup()
}