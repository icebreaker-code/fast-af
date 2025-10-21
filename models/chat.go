package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatWindow struct {
	ID             primitive.ObjectID   `bson:"_id,omitempty" json:"id,omitempty"`
	ParticipantIDs []primitive.ObjectID `bson:"participant_ids" json:"participantIds"`
	IsGroup        bool                 `bson:"is_group" json:"isGroup"`
	CreatedAt      time.Time            `bson:"created_at" json:"createdAt"`
	UpdatedAt      time.Time            `bson:"updated_at" json:"updatedAt"`
}

type Chat struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	ChatWindowID primitive.ObjectID `bson:"chat_window_id" json:"chatWindowId"`
	Msg          string             `bson:"msg" json:"msg"`
	CreatedBy    primitive.ObjectID `bson:"user_id" json:"userId"`
	CreatedAt    time.Time          `bson:"created_at" json:"createdAt"`
}

type ChatRestriction struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	RestrictionType string             `bson:"restriction_type" json:"restrictionType"`
	RestrictedBy    primitive.ObjectID `bson:"restricted_by" json:"restrictedBy"`
}

type ChatConn struct {
	UserID       string
	ChatWindowID string
	Conn         *websocket.Conn
}
