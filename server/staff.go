package server

import (
	"github.com/RemoteState/yourdaily-server/handlers"
	"github.com/RemoteState/yourdaily-server/middlewares"
	"github.com/go-chi/chi"
)

func StaffRoutes(r chi.Router) {
	r.Group(func(staff chi.Router) {
		staff.Use(middlewares.AuthMiddleware)
		staff.Use(middlewares.StaffPermission)
		staff.Put("/location", handlers.PutLocation)

		staff.Route("/order", func(order chi.Router) {
			//get active orders for staff order list
			order.Get("/", handlers.AllActiveOrders)

			//route to get list of new orders
			order.Get("/new", handlers.GetNewOrders)

			//returns a list of all completed orders
			order.Get("/history", handlers.GetOrderHistory)

			//Get order detail of particular order byt id
			order.Get("/{id}", handlers.OrderInfoForStaff)

			//updated the order status to out for delivery
			order.Put("/{id}", handlers.PutOutForDelivery)

			order.Post("/reject/{id}", handlers.RejectOrderForStaff)

		})

		//send response to accept new order
		staff.Put("/accept/order/{id}", handlers.PutAcceptOrder)

		//send otp to verify the delivery of order
		staff.Post("/verify/order/{id}", handlers.PostVerifyOTP)

	})
}
