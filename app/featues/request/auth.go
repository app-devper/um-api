package request

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Login struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	System   string `json:"system" binding:"required"`
}

type Session struct {
	UserId     primitive.ObjectID
	Type       string
	Objective  string
	System     string
	ClientId   string
	ExpireDate time.Time
}
