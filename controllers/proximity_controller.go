package controllers

import (
	"context"
	"time"

	"fast-af/config"
	"fast-af/database"
	"fast-af/models"
	"fast-af/utils"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SetProximityAvailability creates an active proximity entry for a user.
// Expects JSON body with latitude, longitude, radius (meters) and optional expiresInSeconds (int).
func SetProximityAvailability(c *fiber.Ctx) error {
	userID := c.Params("userId")
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	// verify user exists
	exists, err := UserExists(userObjectID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error checking user existence"})
	}
	if !exists {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	// expect a payload that includes latitude, longitude, radius and availabilityId
	var payload struct {
		Latitude       *float64 `json:"latitude"`
		Longitude      *float64 `json:"longitude"`
		Radius         *float64 `json:"radius"`
		AvailabilityID string   `json:"availabilityId"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	if payload.Latitude == nil || payload.Longitude == nil || payload.AvailabilityID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "latitude, longitude and availabilityId are required"})
	}

	// validate coords
	if *payload.Latitude < -90 || *payload.Latitude > 90 || *payload.Longitude < -180 || *payload.Longitude > 180 {
		return c.Status(400).JSON(fiber.Map{"error": "latitude or longitude out of range"})
	}

	// convert availability id to object id
	availObjectID, err := primitive.ObjectIDFromHex(payload.AvailabilityID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid availabilityId"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	// fetch the availability document and ensure it belongs to this user and is available
	var avail models.Availablility
	err = database.DB.Collection("availabilities").FindOne(ctx, bson.M{"_id": availObjectID, "user_id": userObjectID}).Decode(&avail)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Availability not found for user"})
	}
	if !avail.IsAvailable {
		return c.Status(400).JSON(fiber.Map{"error": "Requested availability is not marked available"})
	}

	// compute created and expiry times from availability date + start/end times
	// expected formats: Date = "YYYY-MM-DD", StartTime/EndTime = "HH:MM" or "HH:MM:SS"
	parseLayouts := []string{"2006-01-02 15:04", "2006-01-02 15:04:05"}
	var startTime, endTime time.Time
	var parseErr error
	for _, layout := range parseLayouts {
		startTime, parseErr = time.ParseInLocation(layout, avail.Date+" "+avail.StartTime, time.Local)
		if parseErr == nil {
			break
		}
	}
	if parseErr != nil {
		// fallback to now
		startTime = time.Now()
	}
	parseErr = nil
	for _, layout := range parseLayouts {
		endTime, parseErr = time.ParseInLocation(layout, avail.Date+" "+avail.EndTime, time.Local)
		if parseErr == nil {
			break
		}
	}
	if parseErr != nil {
		endTime = startTime.Add(time.Hour)
	}

	// build proximity doc
	var prox models.ActiveProximity
	prox.UserID = userObjectID
	prox.AvailabilityID = availObjectID
	prox.Latitude = *payload.Latitude
	prox.Longitude = *payload.Longitude
	if payload.Radius != nil {
		prox.Radius = *payload.Radius
	}
	prox.CreatedAt = startTime
	prox.ExpiresAt = endTime

	// availability existence/ownership was verified above

	// prevent creating another active proximity if user already has one
	existingCount, err := database.DB.Collection("active_proximities").CountDocuments(ctx, bson.M{"user_id": userObjectID, "expires_at": bson.M{"$gt": time.Now()}})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error checking existing proximity"})
	}
	if existingCount > 0 {
		return c.Status(400).JSON(fiber.Map{"error": "User already has an active proximity entry"})
	}

	_, err = database.DB.Collection("active_proximities").InsertOne(ctx, prox)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error creating proximity entry"})
	}

	return c.Status(201).JSON(prox)
}

// ToggleProximityOff expires the active proximity entry for a user (sets ExpiresAt to now).
func ToggleProximityOff(c *fiber.Ctx) error {
	userID := c.Params("userId")
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userObjectID, "expires_at": bson.M{"$gt": time.Now()}}
	update := bson.M{"$set": bson.M{"expires_at": time.Now()}}

	res, err := database.DB.Collection("active_proximities").UpdateMany(ctx, filter, update)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error expiring proximity entries"})
	}
	if res.MatchedCount == 0 {
		return c.Status(404).JSON(fiber.Map{"message": "No active proximity entries found"})
	}

	return c.Status(200).JSON(fiber.Map{"message": "Proximity availability expired"})
}

// GetAllActiveProximities returns all active proximity entries (not expired).
func GetAllActiveProximities(c *fiber.Ctx) error {
	var proximities []models.ActiveProximity

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	cursor, err := database.DB.Collection("active_proximities").Find(ctx, bson.M{"expires_at": bson.M{"$gt": time.Now()}})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error fetching active proximities"})
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var p models.ActiveProximity
		cursor.Decode(&p)
		proximities = append(proximities, p)
	}

	return c.JSON(proximities)
}

// GetNearbyUsers returns all active users within the requesting user's radius.
func GetNearbyUsers(c *fiber.Ctx) error {
	userID := c.Params("userId")
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	// Find the latest active proximity for the user
	filterMe := bson.M{"user_id": userObjectID, "expires_at": bson.M{"$gt": time.Now()}}
	var me models.ActiveProximity
	err = database.DB.Collection("active_proximities").FindOne(ctx, filterMe).Decode(&me)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Active proximity for user not found"})
	}

	// fetch other active proximities
	cursor, err := database.DB.Collection("active_proximities").Find(ctx, bson.M{"user_id": bson.M{"$ne": userObjectID}, "expires_at": bson.M{"$gt": time.Now()}})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error fetching nearby proximities"})
	}
	defer cursor.Close(ctx)

	var nearby []struct {
		UserID    primitive.ObjectID `json:"userId" bson:"user_id"`
		Latitude  float64            `json:"latitude" bson:"latitude"`
		Longitude float64            `json:"longitude" bson:"longitude"`
		Radius    float64            `json:"radius" bson:"radius"`
		Distance  float64            `json:"distanceMeters"`
		ExpiresAt time.Time          `json:"expiresAt" bson:"expires_at"`
	}

	for cursor.Next(ctx) {
		var other models.ActiveProximity
		cursor.Decode(&other)
		d := utils.HaversineDistance(me.Latitude, me.Longitude, other.Latitude, other.Longitude)
		// consider within user's radius
		if d <= me.Radius {
			nearby = append(nearby, struct {
				UserID    primitive.ObjectID `json:"userId" bson:"user_id"`
				Latitude  float64            `json:"latitude" bson:"latitude"`
				Longitude float64            `json:"longitude" bson:"longitude"`
				Radius    float64            `json:"radius" bson:"radius"`
				Distance  float64            `json:"distanceMeters"`
				ExpiresAt time.Time          `json:"expiresAt" bson:"expires_at"`
			}{
				UserID:    other.UserID,
				Latitude:  other.Latitude,
				Longitude: other.Longitude,
				Radius:    other.Radius,
				Distance:  d,
				ExpiresAt: other.ExpiresAt,
			})
		}
	}

	return c.Status(200).JSON(nearby)
}

// UpdateProximityLocation updates the active proximity entry's latitude/longitude
// and optionally radius and expiresAt. Expects JSON body with latitude and longitude.
func UpdateProximityLocation(c *fiber.Ctx) error {
	userID := c.Params("userId")
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	// verify user exists
	exists, err := UserExists(userObjectID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error checking user existence"})
	}
	if !exists {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	var payload struct {
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
		Radius    *float64 `json:"radius,omitempty"`
		// ExpiresAt is intentionally NOT accepted here. Expiry is tied to the availability referenced by the proximity.
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	if payload.Latitude == nil || payload.Longitude == nil {
		return c.Status(400).JSON(fiber.Map{"error": "latitude and longitude are required"})
	}
	// basic validation
	if *payload.Latitude < -90 || *payload.Latitude > 90 || *payload.Longitude < -180 || *payload.Longitude > 180 {
		return c.Status(400).JSON(fiber.Map{"error": "latitude or longitude out of range"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userObjectID, "expires_at": bson.M{"$gt": time.Now()}}

	set := bson.M{
		"latitude":  *payload.Latitude,
		"longitude": *payload.Longitude,
	}
	if payload.Radius != nil {
		set["radius"] = *payload.Radius
	}

	update := bson.M{"$set": set}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updated models.ActiveProximity
	err = database.DB.Collection("active_proximities").FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Active proximity not found or could not be updated"})
	}

	return c.Status(200).JSON(updated)
}
