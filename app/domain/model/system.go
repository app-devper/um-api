package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type System struct {
	Id          primitive.ObjectID `bson:"_id" json:"id"`
	ClientId    string             `bson:"clientId" json:"clientId"`
	SystemName  string             `bson:"systemName" json:"systemName"`
	SystemCode  string             `bson:"systemCode" json:"systemCode"`
	Host        string             `bson:"host" json:"host"`
	CreatedBy   primitive.ObjectID `bson:"createdBy" json:"createdBy"`
	CreatedDate time.Time          `bson:"createdDate" json:"createdDate"`
	UpdatedBy   primitive.ObjectID `bson:"updatedBy" json:"updatedBy"`
	UpdatedDate time.Time          `bson:"updatedDate" json:"updatedDate"`
}
