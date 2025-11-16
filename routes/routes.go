package routes

import (
	"fast-af/controllers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func SetupRoutes(app *fiber.App) {
	api := app.Group("/api/v1")

	// generic routes
	api.Get("/ping", controllers.Ping)

	// user routes
	api.Get("/users", controllers.GetUsers)
	api.Get("/users/:id", controllers.GetUserByID)
	api.Patch("/users/:userId", controllers.UpdateUserByID)

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

	// proximity routes
	api.Post("/users/proximity/:userId", controllers.SetProximityAvailability)
	api.Post("/users/proximity/off/:userId", controllers.ToggleProximityOff)
	api.Patch("/users/proximity/:userId", controllers.UpdateProximityLocation)
	api.Get("/proximities/active", controllers.GetAllActiveProximities)
	api.Get("/users/proximity/nearby/:userId", controllers.GetNearbyUsers)

	api.Get("/users/future-availability/:userId", controllers.GetFutureAvailabilityForUser)
	api.Post("/users/future-availability/:userId", controllers.SetFutureAvailability)
	api.Delete("/users/future-availability/:userId/:id", controllers.CancelFutureAvailability)

	// meeting request routes
	api.Post("/users/:targetUserId/meeting-requests", controllers.CreateMeetingRequest)
	api.Get("/users/:userId/meeting-requests", controllers.GetMeetingRequestsForUser)
	api.Get("/users/:userId/sent-meeting-requests", controllers.GetSentMeetingRequestsForUser)
	api.Patch("/meeting-requests/:id", controllers.UpdateMeetingRequestStatus)
	// only the requester can cancel a meeting request
	api.Delete("/meeting-requests/:id", controllers.CancelMeetingRequest)

	// users matching interests
	api.Get("/users-match-interests", controllers.GetUsersByInterests)
	api.Get("/users-match-interests/:userId", controllers.GetUsersByInterests)

	// chat routes (WebSocket and REST fallback)
	api.Get("/chat/ws/:userId", func(c *fiber.Ctx) error {
		userId := c.Params("userId")
		chatWindowId := c.Query("chatWindowId")
		return websocket.New(func(conn *websocket.Conn) {
			controllers.HandleChatWebSocket(conn, userId, chatWindowId)
		})(c)
	})
	api.Post("/chat/window", controllers.CreateChatWindow)
	api.Post("/chat/message", controllers.SendMessage)
	api.Delete("/chat/message/:msgId", controllers.DeleteMessage)
	api.Post("/chat/block", controllers.BlockChat)

	// new chat window/message fetch APIs
	api.Get("/chat/window/:userId", controllers.GetChatWindowsForUser)
	api.Get("/chat/messages/:chatWindowId", controllers.GetMessagesForChatWindow)
}
