package route

import (
	"ChronoverseAPI/internal/controller"
	"ChronoverseAPI/internal/middleware"
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
		})
	})
}
