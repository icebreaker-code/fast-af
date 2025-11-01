package controllers

import (
	"context"
	"log"
	"time"

	"fast-af/config"
	"fast-af/database"
	"fast-af/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ChatConn tracks a user's websocket connection in a chat window
type ChatConn struct {
	UserID       string
	ChatWindowID string
	Conn         *websocket.Conn
}

// chatWindowID -> list of *ChatConn
var chatWindowClients = make(map[string][]*ChatConn)

// WebSocket handler logic for chat window
func HandleChatWebSocket(conn *websocket.Conn, userId string, chatWindowId string) {
	chatConn := &ChatConn{UserID: userId, ChatWindowID: chatWindowId, Conn: conn}
	// Register connection
	chatWindowClients[chatWindowId] = append(chatWindowClients[chatWindowId], chatConn)
	defer func() {
		// Remove connection on close
		var updated []*ChatConn
		for _, cc := range chatWindowClients[chatWindowId] {
			if cc.Conn != conn {
				updated = append(updated, cc)
			}
		}
		chatWindowClients[chatWindowId] = updated
		conn.Close()
	}()

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			break
		}
		// Fetch valid participants from DB
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
		var chatWindow models.ChatWindow
		chatWindowObjID, err := primitive.ObjectIDFromHex(chatWindowId)
		if err != nil {
			log.Println("Invalid chatWindowId:", err)
			cancel()
			continue
		}
		err = database.DB.Collection("chat_windows").FindOne(ctx, bson.M{"_id": chatWindowObjID}).Decode(&chatWindow)
		cancel()
		if err != nil {
			log.Println("DB error fetching chat window:", err)
			continue
		}
		validParticipants := make(map[string]bool)
		for _, pid := range chatWindow.ParticipantIDs {
			validParticipants[pid.Hex()] = true
		}
		// Broadcast only to valid participants
		for _, cc := range chatWindowClients[chatWindowId] {
			if cc.Conn != conn && validParticipants[cc.UserID] {
				if err := cc.Conn.WriteMessage(mt, msg); err != nil {
					log.Println("broadcast error:", err)
				}
			}
		}
	}
}

func ChatWebSocket(c *fiber.Ctx) error {
	userId := c.Params("userId")
	chatWindowId := c.Query("chatWindowId")
	if userId == "" || chatWindowId == "" {
		return c.Status(400).SendString("userId and chatWindowId required as params")
	}
	return websocket.New(func(conn *websocket.Conn) {
		HandleChatWebSocket(conn, userId, chatWindowId)
	})(c)
}

// Create a new chat window (group or 1-1)
func CreateChatWindow(c *fiber.Ctx) error {
	var req struct {
		ParticipantIDs []string `json:"participantIds"`
		IsGroup        bool     `json:"isGroup"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}
	if len(req.ParticipantIDs) < 2 {
		return c.Status(400).JSON(fiber.Map{"error": "At least 2 participants required"})
	}
	var pids []primitive.ObjectID
	for _, id := range req.ParticipantIDs {
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid participant ID: " + id})
		}
		pids = append(pids, oid)
	}
	now := time.Now()
	chatWindow := models.ChatWindow{
		ParticipantIDs: pids,
		IsGroup:        req.IsGroup,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()
	res, err := database.DB.Collection("chat_windows").InsertOne(ctx, chatWindow)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error creating chat window"})
	}
	chatWindow.ID = res.InsertedID.(primitive.ObjectID)
	return c.Status(201).JSON(chatWindow)
}

// Send a new message (WebSocket recommended, but REST fallback)
func SendMessage(c *fiber.Ctx) error {
	var req struct {
		ChatWindowID string `json:"chatWindowId"`
		Msg          string `json:"msg"`
		UserID       string `json:"userId"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}
	chatWindowObjID, err := primitive.ObjectIDFromHex(req.ChatWindowID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid chatWindowId"})
	}
	userID, err := primitive.ObjectIDFromHex(req.UserID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid userId"})
	}
	chat := models.Chat{
		ChatWindowID: chatWindowObjID,
		Msg:          req.Msg,
		CreatedBy:    userID,
		CreatedAt:    time.Now(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()
	res, err := database.DB.Collection("chats").InsertOne(ctx, chat)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error sending message"})
	}
	chat.ID = res.InsertedID.(primitive.ObjectID)
	return c.Status(201).JSON(chat)
}

// Delete a message
func DeleteMessage(c *fiber.Ctx) error {
	msgID := c.Params("msgId")
	oid, err := primitive.ObjectIDFromHex(msgID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid message ID"})
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()
	res, err := database.DB.Collection("chats").DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil || res.DeletedCount == 0 {
		return c.Status(500).JSON(fiber.Map{"error": "Error deleting message or not found"})
	}
	return c.Status(200).JSON(fiber.Map{"message": "Message deleted"})
}

// Block a chat (add restriction)
func BlockChat(c *fiber.Ctx) error {
	var req struct {
		ChatWindowID    string `json:"chatWindowId"`
		RestrictedBy    string `json:"restrictedBy"`
		RestrictionType string `json:"restrictionType"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}
	_, err := primitive.ObjectIDFromHex(req.ChatWindowID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid chatWindowId"})
	}
	restrictedBy, err := primitive.ObjectIDFromHex(req.RestrictedBy)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid restrictedBy"})
	}
	restriction := models.ChatRestriction{
		RestrictionType: req.RestrictionType,
		RestrictedBy:    restrictedBy,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()
	_, err = database.DB.Collection("chat_restrictions").InsertOne(ctx, restriction)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error blocking chat"})
	}
	return c.Status(201).JSON(fiber.Map{"message": "Chat blocked"})
}

// Fetch all chat windows for a user
func GetChatWindowsForUser(c *fiber.Ctx) error {
	userId := c.Params("userId")
	oid, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid userId"})
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()
	cursor, err := database.DB.Collection("chat_windows").Find(ctx, bson.M{
		"participant_ids": bson.M{"$in": []primitive.ObjectID{oid}},
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error fetching chat windows"})
	}
	var chatWindows []models.ChatWindow
	if err := cursor.All(ctx, &chatWindows); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error decoding chat windows"})
	}
	return c.Status(200).JSON(chatWindows)
}

// Fetch all messages for a chat window
func GetMessagesForChatWindow(c *fiber.Ctx) error {
	chatWindowId := c.Params("chatWindowId")
	oid, err := primitive.ObjectIDFromHex(chatWindowId)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid chatWindowId"})
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.DefaultDBContextTimeout)*time.Second)
	defer cancel()
	cursor, err := database.DB.Collection("chats").Find(ctx, bson.M{"chat_window_id": oid})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error fetching messages"})
	}
	var messages []models.Chat
	if err := cursor.All(ctx, &messages); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error decoding messages"})
	}
	return c.Status(200).JSON(messages)
}
