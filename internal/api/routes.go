package api

import (
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

func (api *Api) BindRoutes() {
	api.Router.Use(
		middleware.RequestID,
		middleware.Recoverer,
		middleware.Logger,
		api.Sessions.LoadAndSave,
	)

	api.Router.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Route("/users", func(r chi.Router) {
				r.Post("/signup", api.handleSignupUser)
				r.Post("/sign", api.handleSignUser)
				r.With(api.AuthMiddleware).Post("/logout", api.handleLogoutUser)

			})
		})
	})

}
