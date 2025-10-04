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
}
