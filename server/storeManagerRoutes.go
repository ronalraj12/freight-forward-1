package server

import (
	"github.com/RemoteState/yourdaily-server/handlers"
	"github.com/RemoteState/yourdaily-server/middlewares"
	"github.com/go-chi/chi"
)

func storeMangerRoutes(r chi.Router) {
	r.Group(func(sm chi.Router) {
		sm.Use(middlewares.AuthMiddlewareForSm)

		//dashboard
		sm.Get("/dashboard/stats", handlers.DashBoardStats)
		sm.Get("/dashboard/nsg/{days}", handlers.GetOrderTypeGraphData)
		sm.Get("/dashboard/adg/{days}", handlers.GetOrderAcceptGraphData)
		sm.Get("/dashboard/staff/{staffType}", handlers.GetAllStaffStats)
		sm.Get("/dashboard/staff/{staffType}/{id}", handlers.GetStaffStatsByID)
		sm.Get("/dashboard/user/details", handlers.GetAllUserStats)
		sm.Get("/dashboard/user/details/{id}", handlers.GetUserStatsByID)
		sm.Get("/dashboard/order/disputed", handlers.GetAllDisputedOrders)
		sm.Get("/dashboard/order/disputed/{id}", handlers.GetAllDisputedOrderInfo)
		sm.Put("/dashboard/order/disputed/{id}", handlers.MarkAsResolved)
		sm.Get("/dashboard/order/{orderType}", handlers.GetAllOrdersWithStatus)
		sm.Get("/dashboard/order/new", handlers.GetNewOrderForStoreManger)
		sm.Put("/dashboard/unflag/user/{id}", handlers.UnFlagUser)
		sm.Post("/dashboard/order/history", handlers.GetOrders)
		sm.Get("/dashboard/order/active", handlers.GetOngoingOrder)
		sm.Put("/staff/{status}/{id}", handlers.EnableDisableStaff)
		sm.Put("/staff/update/role", handlers.ChangeStaffRole)
		sm.Get("/scheduled/orders", handlers.GetScheduledOrders)
		sm.Delete("/cancel/scheduled/order/{id}", handlers.CancelScheduledOrder)

		sm.Route("/download", func(smd chi.Router) {
			smd.Post("/scheduled/orders", handlers.DownloadScheduledOrders)
			smd.Post("/user/stats", handlers.DownloadUserStats)
			smd.Post("/order/history", handlers.DownloadOrderHistory)

		})

		// items
		sm.Route("/item", func(item chi.Router) {
			item.Get("/", handlers.GetItemsForStoreManager)
			item.Post("/", handlers.CreateItem)
			item.Get("/{id}", handlers.GetItemById)
			item.Put("/{id}", handlers.ModifyItem)
			item.Delete("/{id}", handlers.ArchiveItem)
			item.Post("/image/{id}", handlers.AddImageForExistingItem)
		})

		// category
		sm.Route("/category", func(category chi.Router) {
			category.Get("/", handlers.GetAllCategories)
			category.Post("/", handlers.CreateCategory)
			category.Put("/{id}", handlers.ModifyCategory)
			category.Delete("/{id}", handlers.ArchiveCategory)
		})

		// staff & order manage
		sm.Route("/staff", func(staff chi.Router) {
			// staff manage
			staff.Post("/", handlers.ApproveStaff)
			staff.Post("/{id}", handlers.RejectStaff)
			staff.Delete("/{id}", handlers.ArchiveStaff)
			staff.Get("/", handlers.GetUnapprovedStaff)

			staff.Get("/{orderId}", handlers.GetNearbyStaffList)
			staff.Post("/assign", handlers.AssignOrderToStaff)
		})

		// image upload
		sm.Post("/image/{imageType}", handlers.AddImageOfGivenType)

		// discount offers
		sm.Post("/offer", handlers.CreateNewOffer)
		sm.Get("/offer", handlers.GetActiveOffer)
		sm.Delete("/offer", handlers.ArchiveActiveOffer)
	})
}
