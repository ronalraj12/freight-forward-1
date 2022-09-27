package server

import (
	"github.com/RemoteState/yourdaily-server/handlers"
	"github.com/RemoteState/yourdaily-server/middlewares"
	"github.com/go-chi/chi"
)

func userRoutes(r chi.Router) {
	r.Group(func(user chi.Router) {
		user.Use(middlewares.AuthMiddleware)

		// user info
		user.Get("/", handlers.GetUserInfo)
		user.Put("/", handlers.UpdateUserInfo)

		// address
		user.Route("/address", func(address chi.Router) {
			address.Post("/", handlers.PostAddress)
			address.Get("/", handlers.GetAllAddress)
			address.Get("/{id}", handlers.GetAddressByID)
			address.Put("/{id}", handlers.PutAddress)
			address.Delete("/{id}", handlers.ArchiveAddress)
		})

		user.Route("/order", func(order chi.Router) {
			// order - now
			order.Post("/now", handlers.OrderNow)
			order.Get("/", handlers.AllPastOrder)

			//get order info by id
			order.Get("/{id}", handlers.OrderInfo)
			order.Delete("/{id}", handlers.CancelOrder)
			order.Get("/status/{id}", handlers.OrderStatus)
			order.Get("/staff/{id}", handlers.GetStaffByID)

			//routes for list of active order of the users active == processing,accepted,outForDelivery
			order.Get("/active", handlers.GetActiveOrdersForUser)

			//route to mark order as disputed
			order.Post("/dispute/{id}", handlers.DisputeOrder)

			//confirm order completion
			order.Put("/confirm/{id}", handlers.ConfirmOrderDelivery)

			// schedule order routes
			order.Route("/schedule", func(schedule chi.Router) {
				schedule.Post("/", handlers.PostScheduledOrder)
				schedule.Get("/", handlers.GetAllScheduledOrders)
				schedule.Get("/{id}", handlers.GetScheduledOrder)
				schedule.Delete("/{id}", handlers.ArchiveScheduledOrder)
			})

		})

		// item & category
		user.Get("/item", handlers.GetAllItems)
		user.Get("/category", handlers.GetAllCategories)

		// fcm
		user.Post("/fcm", handlers.UpdateFcmToken)

		// profile image
		user.Post("/profile", handlers.UploadImage)

		// offer
		user.Get("/offer", handlers.GetActiveOffer)

		// discount
		user.Get("/discount", handlers.GetActiveDiscount)
		//admin contact info
		user.Get("/admin/contact", handlers.GetAdminContactInfo)

	})
}
