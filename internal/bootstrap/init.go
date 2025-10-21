package bootstrap

import (
	"chrononewsapi/internal/adapter"
	"chrononewsapi/internal/config"
	"chrononewsapi/internal/handler/controller"
	"chrononewsapi/internal/handler/middleware"
	"chrononewsapi/internal/repository"
	"chrononewsapi/internal/service"
	"chrononewsapi/internal/service/compression"
	"chrononewsapi/internal/service/queue"
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

func Init(app *chi.Mux, db *gorm.DB, config *config.Config, validator *validator.Validate, client *http.Client, ctx context.Context) {
	userRepository := repository.NewUserRepository()
	categoryRepository := repository.NewCategoryRepository()
	fileRepository := repository.NewFileRepository()
	postRepository := repository.NewPostRepository()
	resetRepository := repository.NewResetRepository()

	storageAdapter := adapter.NewStorageAdapter()
	captchaAdapter := adapter.NewCaptchaAdapter(client)
	emailAdapter := adapter.NewEmailAdapter()

	compressionService := compression.NewCompressionService(db, config)

	var rabbitMQService *queue.RabbitMQService
	if config.RabbitMQ.Enabled {
		var err error
		rabbitMQService, err = queue.NewRabbitMQService(
			config.RabbitMQ.URL,
			config.RabbitMQ.QueueName,
		)
		if err != nil {
			slog.Error("failed to connect to rabbitmq", "error", err)
			slog.Warn("rabbitmq not available, falling back to synchronous compression")
		} else {
			batchTimeout := time.Duration(config.RabbitMQ.BatchTimeout) * time.Second

			handler := func(fileIDs []int32) ([]queue.FileProcessResult, error) {
				result, err := compressionService.ProcessFiles(context.Background(), fileIDs)
				if err != nil {
					return nil, err
				}

				queueResults := make([]queue.FileProcessResult, len(result.FileResults))
				for i, fr := range result.FileResults {
					queueResults[i] = queue.FileProcessResult{
						FileID:  fr.FileID,
						Success: fr.Success,
						Error:   fr.Error,
					}
				}
				return queueResults, nil
			}

			err = rabbitMQService.StartConsumer(
				ctx,
				config.RabbitMQ.BatchSize,
				batchTimeout,
				handler,
			)
			if err != nil {
				slog.Error("failed to start rabbitmq consumer", "error", err)
				rabbitMQService = nil
			} else {
				slog.Info("rabbitmq consumer started",
					"batch_size", config.RabbitMQ.BatchSize,
					"timeout", batchTimeout)
			}
		}
	}

	userService := service.NewUserService(db, userRepository, postRepository, resetRepository, storageAdapter, captchaAdapter, emailAdapter, validator, config)
	categoryService := service.NewCategoryService(db, categoryRepository, userRepository, postRepository, validator)
	postService := service.NewPostService(db, postRepository, userRepository, fileRepository, categoryRepository, storageAdapter, compressionService, rabbitMQService, validator, config)
	resetService := service.NewResetService(db, resetRepository, userRepository, emailAdapter, captchaAdapter, validator, config)
	fileService := service.NewFileService(db, fileRepository, storageAdapter, config, validator)

	userController := controller.NewUserController(userService)
	categoryController := controller.NewCategoryController(categoryService)
	postController := controller.NewPostController(postService)
	resetController := controller.NewResetController(resetService)
	fileController := controller.NewFileController(fileService)

	userMiddleware := middleware.NewUserMiddleware(userService)

	router := Route{
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
