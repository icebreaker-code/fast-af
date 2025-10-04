package controllers

import (
	"context"
	"time"

	"fast-af/config"
	"fast-af/database"
	"fast-af/models"

	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var googleOauthConfig = &config.GoogleOauthConfig

// Handler to start Google OAuth login
func GoogleLogin(c *fiber.Ctx) error {
	url := googleOauthConfig.AuthCodeURL("randomstate")
	return c.Redirect(url)
}

// Handler for Google OAuth callback
func GoogleCallback(c *fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		return c.Status(400).SendString("Code not found")
	}

	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return c.Status(500).SendString("Failed to exchange token")
	}

	client := googleOauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return c.Status(500).SendString("Failed to get user info")
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	var userInfo struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	json.Unmarshal(data, &userInfo)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	// Check if user already exists
	var existingUser models.User
	err = database.DB.Collection("users").FindOne(
		ctx,
		bson.M{"email": userInfo.Email},
		options.FindOne().SetProjection(bson.M{"_id": 0}),
	).Decode(&existingUser)
	if err == nil {
		// User exists, do not update
		return c.Status(200).SendString(fmt.Sprintf("Welcome back, %s!", existingUser.Name))
	}
	if err.Error() != "mongo: no documents in result" {
		// Some other error
		fmt.Println(err)
		return c.Status(500).SendString("Failed to check user existence")
	}

	// User does not exist, create new user
	newUser := models.User{
		Name:              userInfo.Name,
		Email:             userInfo.Email,
		ProfilePictureURL: userInfo.Picture,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	_, err = database.DB.Collection("users").InsertOne(ctx, newUser)
	if err != nil {
		return c.Status(500).SendString("Failed to create user")
	}

	return c.Status(201).SendString(fmt.Sprintf("Welcome, %s!", userInfo.Name))
}

func GetUsers(c *fiber.Ctx) error {
	var users []models.User

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	cursor, err := database.DB.Collection("users").Find(ctx, bson.M{})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error fetching users"})
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var user models.User
		cursor.Decode(&user)
		users = append(users, user)
	}

	return c.JSON(users)
}

func CreateUser(c *fiber.Ctx) error {
	var user models.User
	if err := c.BodyParser(&user); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	_, err := database.DB.Collection("users").InsertOne(ctx, user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error creating user"})
	}

	return c.Status(201).JSON(user)
}

func GetUserByID(c *fiber.Ctx) error {
	userID := c.Params("id")
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	var user models.User

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	err = database.DB.Collection("users").FindOne(ctx, bson.M{"_id": userObjectID}).Decode(&user)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(user)
}

func UserExists(userId primitive.ObjectID) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()
	count, err := database.DB.Collection("users").CountDocuments(ctx, bson.M{"_id": userId})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
