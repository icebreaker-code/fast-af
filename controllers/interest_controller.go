package controllers

import (
	"context"
	"fmt"
	"time"

	"fast-af/config"
	"fast-af/database"
	"fast-af/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetAllInterests(c *fiber.Ctx) error {
	var interests []models.Interest
	cursor, err := database.DB.Collection("interests").Find(nil, bson.M{})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error fetching interests"})
	}
	defer cursor.Close(nil)

	for cursor.Next(nil) {
		var interest models.Interest
		cursor.Decode(&interest)
		interests = append(interests, interest)
	}

	return c.JSON(interests)
}

func CreateInterest(c *fiber.Ctx) error {
	var interest models.Interest
	if err := c.BodyParser(&interest); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	// return error if interest with same name exists or if name is empty
	if interest.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Interest name cannot be empty"})
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()
	count, err := database.DB.Collection("interests").CountDocuments(ctx, bson.M{"name": interest.Name})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error checking existing interests"})
	}
	if count > 0 {
		return c.Status(400).JSON(fiber.Map{"error": "Interest with this name already exists"})
	}

	_, err = database.DB.Collection("interests").InsertOne(nil, interest)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error creating interest"})
	}

	return c.Status(201).JSON(interest)
}

func RemoveInterest(c *fiber.Ctx) error {
	interestID := c.Params("id")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()
	_, err := database.DB.Collection("interests").DeleteOne(ctx, bson.M{"_id": interestID})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error deleting interest"})
	}

	return c.Status(200).JSON(fiber.Map{"message": "Interest deleted"})
}

func InterestExists(interestID primitive.ObjectID) (bool, error) {
	count, err := database.DB.Collection("interests").CountDocuments(nil, bson.M{"_id": interestID})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func AddUserInterests(c *fiber.Ctx) error {
	var userInterests []models.UserInterest
	if err := c.BodyParser(&userInterests); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	userID := userInterests[0].UserID

	// check if user exists
	exists, err := UserExists(userID)
	if err != nil {
		fmt.Println("Error checking user existence:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Error checking user existence"})
	}
	if !exists {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	// check if all interests exist
	for _, ui := range userInterests {
		exists, err := InterestExists(ui.InterestID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Error checking interest existence"})
		}
		if !exists {
			return c.Status(404).JSON(fiber.Map{"error": fmt.Sprintf("Interest not found: %s", ui.InterestID)})
		}
	}

	// check if user already has any of the interests
	for _, ui := range userInterests {
		count, err := database.DB.Collection("user_interests").CountDocuments(nil, bson.M{"user_id": userID, "interest_id": ui.InterestID})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Error checking existing user interests"})
		}
		if count > 0 {
			return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("User already has interest: %s", ui.InterestID)})
		}
	}

	// insert user interests
	var docs []interface{}
	for _, ui := range userInterests {
		docs = append(docs, ui)
	}

	_, err = database.DB.Collection("user_interests").InsertMany(nil, docs)
	if err != nil {
		fmt.Println("Error inserting user interests:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Error adding user interests"})
	}

	return c.Status(201).JSON(userInterests)
}

func GetUserInterests(c *fiber.Ctx) error {
	var userInterests []models.UserInterest
	userID := c.Params("userId")
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	cursor, err := database.DB.Collection("user_interests").Find(ctx, bson.M{"user_id": userObjectID})
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Error fetching user interests"})
	}
	defer cursor.Close(nil)

	for cursor.Next(nil) {
		var Interest models.UserInterest
		cursor.Decode(&Interest)
		userInterests = append(userInterests, Interest)
	}
	return c.JSON(userInterests)
}

func RemoveUserInterest(c *fiber.Ctx) error {
	userID := c.Params("userId")
	interestID := c.Params("interestId")
	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()
	_, err = database.DB.Collection("user_interests").DeleteOne(ctx, bson.M{"user_id": uid, "interest_id": interestID})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error removing user interest"})
	}

	return c.Status(200).JSON(fiber.Map{"message": "Interest removed from user"})
}

func SearchInterestByPattern(pattern string) ([]models.Interest, error) {
	var interests []models.Interest
	filter := bson.M{"name": bson.M{"$regex": pattern, "$options": "i"}}
	cursor, err := database.DB.Collection("interests").Find(nil, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(nil)

	for cursor.Next(nil) {
		var interest models.Interest
		cursor.Decode(&interest)
		interests = append(interests, interest)
	}
	return interests, nil
}

func SearchInterests(c *fiber.Ctx) error {
	pattern := c.Params("pattern", "")
	interests, err := SearchInterestByPattern(pattern)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error searching interests"})
	}

	return c.JSON(interests)
}
