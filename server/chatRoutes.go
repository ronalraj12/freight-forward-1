package server

import (
	"github.com/RemoteState/yourdaily-server/handlers"
	"github.com/RemoteState/yourdaily-server/middlewares"
	"github.com/go-chi/chi"
)

func ChatRoutes(r chi.Router) {
	r.Group(func(chat chi.Router) {
		chat.Use(middlewares.AuthMiddleware)
		chat.Get("/", handlers.GetAllMessage)
		chat.Post("/", handlers.PostMessage)

	})
}
