package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Session struct {
	Id          primitive.ObjectID `bson:"_id" json:"userRefId"`
	UserId      primitive.ObjectID `bson:"userId" json:"-"`
	ClientId    string             `bson:"clientId" json:"clientId"`
	System      string             `bson:"system" json:"system"`
	Type        string             `bson:"type" json:"type"`
	Objective   string             `bson:"objective" json:"objective"`
	CreatedDate time.Time          `bson:"createdDate" json:"createdDate"`
	ExpireDate  time.Time          `bson:"expireDate" json:"expireDate"`
}
