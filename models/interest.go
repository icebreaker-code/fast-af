package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Interest struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name            string             `bson:"name" json:"name"`
	Category        string             `bson:"category" json:"category"`
	Description     string             `bson:"description" json:"description"`
	CreatedByUserID primitive.ObjectID `bson:"created_by_user_id" json:"createdByUserId"`
}
