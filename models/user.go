package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Email             string             `bson:"email" json:"email"`
	PasswordHash      string             `bson:"password_hash" json:"passwordHash"`
	Name              string             `bson:"name" json:"name"`
	Age               int                `bson:"age" json:"age"`
	Gender            string             `bson:"gender" json:"gender"`
	Locality          string             `bson:"locality" json:"locality"`
	ProfilePictureURL string             `bson:"profile_picture_url" json:"profilePictureUrl"`
	Bio               string             `bson:"bio" json:"bio"`
	TrustScore        float64            `bson:"trust_score" json:"trustScore"`
	Verified          bool               `bson:"verified" json:"verified"`
	CreatedAt         time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt         time.Time          `bson:"updated_at" json:"updatedAt"`
}

type UserInterest struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID     primitive.ObjectID `bson:"user_id" json:"userId"`
	InterestID primitive.ObjectID `bson:"interest_id" json:"interestId"`
}
