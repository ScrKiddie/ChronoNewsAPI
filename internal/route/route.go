package route

import (
	"chronoverseapi/internal/controller"
	"chronoverseapi/internal/middleware"
	"github.com/go-chi/chi/v5"
	"net/http"
)

type Route struct {
	App                *chi.Mux
	UserMiddleware     *middleware.UserMiddleware
	UserController     *controller.UserController
	CategoryController *controller.CategoryController
}

func (r *Route) Setup() {
	r.App.Route("/api", func(c chi.Router) {
		c.Group(func(guest chi.Router) {
			guest.Post("/user/register", r.UserController.Register)
			guest.Post("/user/login", r.UserController.Login)
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

			auth.Get("/category", r.CategoryController.List)
			auth.Post("/category", r.CategoryController.Create)
			auth.Get("/category/{id}", r.CategoryController.Get)
			auth.Put("/category/{id}", r.CategoryController.Update)
			auth.Delete("/category/{id}", r.CategoryController.Delete)
		})
	})
	r.App.Handle("/profile_picture/*", http.StripPrefix("/profile_picture/", http.FileServer(http.Dir("./storage/profile_picture/"))))
}
