package server

import (
	"github.com/RemoteState/yourdaily-server/handlers"
	"github.com/RemoteState/yourdaily-server/middlewares"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/RemoteState/yourdaily-server/utils"
	"github.com/go-chi/chi"
	"net/http"
)

type Server struct {
	chi.Router
}

// SetupRoutes provides all the routes that can be used
func SetupRoutes() *Server {
	router := chi.NewRouter()
	router.Route("/api", func(r chi.Router) {
		r.Use(middlewares.CommonMiddlewares()...)

		// health endpoint
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			utils.RespondJSON(w, 200, models.Response{Success: true})
		})

		// public routes
		r.Post("/register", handlers.Register)
		r.Post("/check", handlers.IsPhoneExisting)
		r.Post("/staff-register", handlers.RegisterStaff)
		r.Post("/sm-login", handlers.LoginStoreManager)

		// private routes- user only
		r.Route("/user", func(r chi.Router) {
			r.Group(userRoutes)
		})

		// private routes- store-manager only
		r.Route("/store-manager", func(sm chi.Router) {
			sm.Group(storeMangerRoutes)
		})

		// private routes- cart-boy only
		r.Route("/staff", func(r chi.Router) {
			r.Group(StaffRoutes)
		})
		//chats routes
		r.Route("/chat", func(r chi.Router) {
			r.Group(ChatRoutes)
		})
		//chats

	})
	return &Server{Router: router}
}

func (svc *Server) Run(port string) error {
	return http.ListenAndServe(port, svc)
}
