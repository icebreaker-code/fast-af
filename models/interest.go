package models

type Interest struct {
	ID              string `bson:"_id,omitempty" json:"id"`
	Name            string `bson:"name" json:"name"`
	Category        string `bson:"category" json:"category"`
	Description     string `bson:"description" json:"description"`
	CreatedByUserID string `bson:"created_by_user_id" json:"createdByUserId"`
}
