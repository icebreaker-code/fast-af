package main

import (
	"fast-af/config"
	"fast-af/database"
	"fast-af/routes"
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {

	// load the config
	config.LoadConfig()

	// connect to mongo
	database.ConnectMongo()

	// create a new fiber instance
	app := fiber.New()

	// setup the routes
	routes.SetupRoutes(app)

	log.Fatal(app.Listen(":3000"))
}
