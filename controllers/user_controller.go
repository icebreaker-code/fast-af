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

// PATCH /users/:userId - update user info
func UpdateUserByID(c *fiber.Ctx) error {
	userID := c.Params("userId")
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	var updateData map[string]interface{}
	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	// Remove fields that should not be updated
	delete(updateData, "_id")
	delete(updateData, "createdAt")
	delete(updateData, "email") // Do not allow email change
	updateData["updatedAt"] = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	update := bson.M{"$set": updateData}
	res, err := database.DB.Collection("users").UpdateByID(ctx, userObjectID, update)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error updating user"})
	}
	if res.MatchedCount == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	var updatedUser models.User
	err = database.DB.Collection("users").FindOne(ctx, bson.M{"_id": userObjectID}).Decode(&updatedUser)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error fetching updated user"})
	}

	return c.Status(200).JSON(updatedUser)
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

// Availablility model handling

func SetAvailableNow(c *fiber.Ctx) error {
	userID := c.Params("userId")
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	// Check if user exists
	exists, err := UserExists(userObjectID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to check user existence"})
	}
	if !exists {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	avail := models.Availablility{
		UserID:      userObjectID,
		Date:        time.Now().Format("2006-01-02"),
		StartTime:   time.Now().Format("15:04"),
		EndTime:     "", // Open-ended for now
		IsAvailable: true,
		Location:    c.Query("location", ""),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	_, err = database.DB.Collection("availabilities").InsertOne(ctx, avail)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to set availability"})
	}
	return c.Status(200).JSON(fiber.Map{"message": "User is now available"})
}

func UnsetAvailableNow(c *fiber.Ctx) error {
	userID := c.Params("userId")
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	// Find the most recent 'available now' entry for this user and today
	filter := bson.M{
		"user_id":      userObjectID,
		"is_available": true,
		"date":         time.Now().Format("2006-01-02"),
		"end_time":     "", // Only open-ended (currently available) entries
	}
	// Sort by start_time descending to get the latest
	opts := options.FindOne().SetSort(bson.D{{"start_time", -1}})
	var avail models.Availablility
	err = database.DB.Collection("availabilities").FindOne(ctx, filter, opts).Decode(&avail)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "No active availability found"})
	}

	update := bson.M{"$set": bson.M{"is_available": false, "end_time": time.Now().Format("15:04")}}
	_, err = database.DB.Collection("availabilities").UpdateByID(ctx, avail.ID, update)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to unset availability"})
	}
	return c.Status(200).JSON(fiber.Map{"message": "User is no longer available"})
}

func UserAvailableNow(c *fiber.Ctx) error {
	userID := c.Params("userId")
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	filter := bson.M{
		"user_id":      userObjectID,
		"is_available": true,
		"date":         time.Now().Format("2006-01-02"),
		"end_time":     "", // Only open-ended (currently available) entries
	}
	count, err := database.DB.Collection("availabilities").CountDocuments(ctx, filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to check availability"})
	}
	if count > 0 {
		return c.Status(200).JSON(fiber.Map{"available": true})
	}
	return c.Status(200).JSON(fiber.Map{"available": false})
}

func SetFutureAvailability(c *fiber.Ctx) error {
	userID := c.Params("userId")
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	var avail models.Availablility
	if err := c.BodyParser(&avail); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}
	avail.UserID = userObjectID
	avail.IsAvailable = true

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	_, err = database.DB.Collection("availabilities").InsertOne(ctx, avail)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to set future availability"})
	}
	return c.Status(201).JSON(fiber.Map{"message": "Future availability set"})
}

func GetFutureAvailabilityForUser(c *fiber.Ctx) error {
	userId := c.Params("userId")
	userObjectID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	filter := bson.M{
		"user_id":      userObjectID,
		"is_available": true,
		"end_time":     bson.M{"$ne": ""}, // Exclude 'available now' entries
	}
	cursor, err := database.DB.Collection("availabilities").Find(ctx, filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch future availability"})
	}
	defer cursor.Close(ctx)

	var availabilities []models.Availablility
	for cursor.Next(ctx) {
		var avail models.Availablility
		cursor.Decode(&avail)
		availabilities = append(availabilities, avail)
	}

	return c.Status(200).JSON(availabilities)
}

func CancelFutureAvailability(c *fiber.Ctx) error {
	userID := c.Params("userId")
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}
	var req struct {
		Date      string `json:"date"`
		StartTime string `json:"startTime"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}
	if req.Date == "" || req.StartTime == "" {
		return c.Status(400).JSON(fiber.Map{"error": "date and startTime required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userObjectID, "date": req.Date, "start_time": req.StartTime, "is_available": true}
	res, err := database.DB.Collection("availabilities").DeleteOne(ctx, filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to cancel future availability"})
	}
	if res.DeletedCount == 0 {
		return c.Status(404).JSON(fiber.Map{"message": "No matching future availability found"})
	}
	return c.Status(200).JSON(fiber.Map{"message": "Future availability cancelled"})
}

// POST /users/:targetUserId/meeting-requests
func CreateMeetingRequest(c *fiber.Ctx) error {
	targetUserId := c.Params("targetUserId")
	targetObjectID, err := primitive.ObjectIDFromHex(targetUserId)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid target user ID"})
	}

	var req struct {
		AvailabilityID string `json:"availabilityId"`
		Message        string `json:"message"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}
	availabilityObjectID, err := primitive.ObjectIDFromHex(req.AvailabilityID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid availability ID"})
	}

	// For demo, requesterId from query param (in real app, from auth context)
	requesterId := c.Query("requesterId")
	requesterObjectID, err := primitive.ObjectIDFromHex(requesterId)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid requester ID"})
	}

	meetingReq := models.MeetingRequest{
		RequesterID:    requesterObjectID,
		TargetUserID:   targetObjectID,
		AvailabilityID: availabilityObjectID,
		Message:        req.Message,
		Status:         "pending",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	res, err := database.DB.Collection("meeting_requests").InsertOne(ctx, meetingReq)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create meeting request"})
	}
	meetingReq.ID = res.InsertedID.(primitive.ObjectID)
	return c.Status(201).JSON(meetingReq)
}

// GET /users/:userId/meeting-requests (for target user)
func GetMeetingRequestsForUser(c *fiber.Ctx) error {
	userId := c.Params("userId")
	userObjectID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	filter := bson.M{"target_user_id": userObjectID}
	cursor, err := database.DB.Collection("meeting_requests").Find(ctx, filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch meeting requests"})
	}
	defer cursor.Close(ctx)

	var requests []models.MeetingRequest
	for cursor.Next(ctx) {
		var req models.MeetingRequest
		cursor.Decode(&req)
		requests = append(requests, req)
	}
	return c.Status(200).JSON(requests)
}

// PATCH /meeting-requests/:id (accept/reject)
func UpdateMeetingRequestStatus(c *fiber.Ctx) error {
	reqId := c.Params("id")
	reqObjectID, err := primitive.ObjectIDFromHex(reqId)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid meeting request ID"})
	}

	var body struct {
		Status string `json:"status"` // accepted or rejected
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}
	if body.Status != "accepted" && body.Status != "rejected" {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid status"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{"status": body.Status, "updated_at": time.Now()}}
	res, err := database.DB.Collection("meeting_requests").UpdateByID(ctx, reqObjectID, update)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update meeting request status"})
	}
	if res.MatchedCount == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "Meeting request not found"})
	}

	var updatedReq models.MeetingRequest
	err = database.DB.Collection("meeting_requests").FindOne(ctx, bson.M{"_id": reqObjectID}).Decode(&updatedReq)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch updated meeting request"})
	}
	return c.Status(200).JSON(updatedReq)
}

// DELETE /meeting-requests/:id (cancel by requester)
func CancelMeetingRequest(c *fiber.Ctx) error {
	reqId := c.Params("id")
	reqObjectID, err := primitive.ObjectIDFromHex(reqId)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid meeting request ID"})
	}

	// For demo, requesterId from query param (in real app, from auth context)
	requesterId := c.Query("requesterId")
	requesterObjectID, err := primitive.ObjectIDFromHex(requesterId)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid requester ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	// Only allow update if requester matches
	filter := bson.M{"_id": reqObjectID, "requester_id": requesterObjectID}
	update := bson.M{"$set": bson.M{"status": "deleted", "updated_at": time.Now()}}
	res, err := database.DB.Collection("meeting_requests").UpdateOne(ctx, filter, update)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to cancel meeting request"})
	}
	if res.MatchedCount == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "Meeting request not found or not owned by requester"})
	}
	return c.Status(200).JSON(fiber.Map{"message": "Meeting request marked as deleted"})
}

// GET /users/:userId/sent-meeting-requests
func GetSentMeetingRequestsForUser(c *fiber.Ctx) error {
	userId := c.Params("userId")
	userObjectID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()

	filter := bson.M{"requester_id": userObjectID}
	cursor, err := database.DB.Collection("meeting_requests").Find(ctx, filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch sent meeting requests"})
	}
	defer cursor.Close(ctx)

	var requests []models.MeetingRequest
	for cursor.Next(ctx) {
		var req models.MeetingRequest
		cursor.Decode(&req)
		requests = append(requests, req)
	}
	return c.Status(200).JSON(requests)
}
