package route

import (
	"chrononewsapi/internal/controller"
	"chrononewsapi/internal/middleware"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/spf13/viper"
)

type Route struct {
	App                *chi.Mux
	UserMiddleware     *middleware.UserMiddleware
	UserController     *controller.UserController
	CategoryController *controller.CategoryController
	PostController     *controller.PostController
	ResetController    *controller.ResetController
	FileController     *controller.FileController
	Config             *viper.Viper
}

func (r *Route) Setup() {
	r.App.Route("/api", func(c chi.Router) {
		c.Group(func(guest chi.Router) {
			guest.Post("/user/login", r.UserController.Login)
			guest.Get("/post", r.PostController.Search)
			guest.Get("/post/{id}", r.PostController.Get)
			guest.Patch("/post/{id}/view", r.PostController.IncrementViewCount)
			guest.Get("/category", r.CategoryController.List)
			guest.Post("/reset/request", r.ResetController.RequestResetEmail)
			guest.Patch("/reset", r.ResetController.Reset)
		})

		c.Group(func(auth chi.Router) {
			auth.Use(r.UserMiddleware.Authorize)
			auth.Get("/user/current", r.UserController.Current)
			auth.Patch("/user/current/profile", r.UserController.UpdateProfile)
			auth.Patch("/user/current/password", r.UserController.UpdatePassword)
			auth.Get("/user", r.UserController.Search)
			auth.Post("/user", r.UserController.Create)
			auth.Get("/user/{id}", r.UserController.Get)
			auth.Put("/user/{id}", r.UserController.Update)
			auth.Delete("/user/{id}", r.UserController.Delete)

			auth.Post("/category", r.CategoryController.Create)
			auth.Get("/category/{id}", r.CategoryController.Get)
			auth.Put("/category/{id}", r.CategoryController.Update)
			auth.Delete("/category/{id}", r.CategoryController.Delete)

			auth.Post("/post", r.PostController.Create)
			auth.Put("/post/{id}", r.PostController.Update)
			auth.Delete("/post/{id}", r.PostController.Delete)

			auth.Post("/image", r.FileController.UploadImage)
		})
	})

	profilePicturePath := r.Config.GetString("storage.profile")
	postPicturePath := r.Config.GetString("storage.post")

	r.App.Handle("/profile_picture/*", http.StripPrefix("/profile_picture/", http.FileServer(http.Dir(profilePicturePath))))
	r.App.Handle("/post_picture/*", http.StripPrefix("/post_picture/", http.FileServer(http.Dir(postPicturePath))))
}
