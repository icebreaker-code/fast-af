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
	api.Get("/auth/google/login", controllers.GoogleLogin)
	api.Get("/auth/google/callback", controllers.GoogleCallback)
}
