package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ActiveProximity struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID         primitive.ObjectID `bson:"user_id" json:"userId"`
	AvailabilityID primitive.ObjectID `bson:"availability_id" json:"availabilityId"`
	Latitude       float64            `bson:"latitude" json:"latitude"`
	Longitude      float64            `bson:"longitude" json:"longitude"`
	Radius         float64            `bson:"radius" json:"radius"` // in meters
	CreatedAt      time.Time          `bson:"created_at" json:"createdAt"`
	ExpiresAt      time.Time          `bson:"expires_at" json:"expiresAt"`
}
