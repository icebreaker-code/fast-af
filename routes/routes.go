package routes

import (
	"fast-af/controllers"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	api := app.Group("/api/v1")

	// generic routes
	api.Get("/ping", controllers.Ping)

	// user routes
	api.Get("/users", controllers.GetUsers)
	api.Get("/users/:id", controllers.GetUserByID)

	api.Get("/auth/google/login", controllers.GoogleLogin)
	api.Get("/auth/google/callback", controllers.GoogleCallback)

	// interest routes
	api.Get("/interests", controllers.GetAllInterests)
	api.Post("/interests", controllers.CreateInterest)
	api.Delete("/interests/:id", controllers.RemoveInterest)

	api.Get("/users/interests/:userId", controllers.GetUserInterests)
	api.Post("/users/interests", controllers.AddUserInterests)
	api.Delete("/users/interests/:userId/:interestId", controllers.RemoveUserInterest)

	api.Get("/interests/matches/:pattern", controllers.SearchInterests)

	// availability routes
	api.Get("/users/available-now/:userId", controllers.UserAvailableNow)
	api.Post("/users/available-now/:userId", controllers.SetAvailableNow)
	api.Post("/users/unset-available-now/:userId", controllers.UnsetAvailableNow)

	api.Get("/users/future-availability/:userId", controllers.GetFutureAvailabilityForUser)
	api.Post("/users/future-availability/:userId", controllers.SetFutureAvailability)
	api.Delete("/users/future-availability/:userId/:id", controllers.CancelFutureAvailability)

	// chat routes (WebSocket and REST fallback)
	api.Get("/chat/ws/:userId", websocket.New(controllers.ChatWebSocket))
	api.Post("/chat/window", controllers.CreateChatWindow)
	api.Post("/chat/message", controllers.SendMessage)
	api.Delete("/chat/message/:msgId", controllers.DeleteMessage)
	api.Post("/chat/block", controllers.BlockChat)
}
