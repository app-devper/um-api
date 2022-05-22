package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type User struct {
	Id          primitive.ObjectID `bson:"_id" json:"id"`
	FirstName   string             `bson:"firstName" json:"firstName"`
	LastName    string             `bson:"lastName" json:"lastName"`
	Username    string             `bson:"username" json:"username"`
	ClientId    string             `bson:"clientId" json:"clientId"`
	Password    string             `bson:"password" json:"-"`
	Role        string             `bson:"role" json:"role"`
	Status      string             `bson:"status" json:"status"`
	Phone       string             `bson:"phone" json:"phone"`
	Email       string             `bson:"email" json:"email"`
	CreatedBy   primitive.ObjectID `bson:"createdBy" json:"createdBy"`
	CreatedDate time.Time          `bson:"createdDate" json:"createdDate"`
	UpdatedBy   primitive.ObjectID `bson:"updatedBy" json:"updatedBy"`
	UpdatedDate time.Time          `bson:"updatedDate" json:"updatedDate"`
}
