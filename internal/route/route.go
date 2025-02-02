package route

import (
	"chronoverseapi/internal/controller"
	"chronoverseapi/internal/middleware"
	"github.com/go-chi/chi/v5"
)

type Route struct {
	App            *chi.Mux
	UserMiddleware *middleware.UserMiddleware
	UserController *controller.UserController
}

func (r *Route) Setup() {
	r.App.Route("/api/user", func(c chi.Router) {
		c.Group(func(guest chi.Router) {
			guest.Post("/register", r.UserController.Register)
			guest.Post("/login", r.UserController.Login)
		})

		c.Group(func(auth chi.Router) {
			auth.Use(r.UserMiddleware.Authorize)
			auth.Get("/current", r.UserController.Current)
			auth.Patch("/current/profile", r.UserController.UpdateProfile)
			auth.Patch("/current/password", r.UserController.UpdatePassword)
			auth.Get("/", r.UserController.Search)
			auth.Post("/", r.UserController.Create)
			auth.Get("/{id}", r.UserController.Get)
			auth.Put("/{id}", r.UserController.Update)
			auth.Delete("/{id}", r.UserController.Delete)
		})
	})
}
