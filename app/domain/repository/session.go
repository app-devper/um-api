package repository

import (
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
	"um/app/core/utils"
	"um/app/domain/model"
	"um/app/featues/request"
	"um/db"
)

type sessionEntity struct {
	sessionRepo *mongo.Collection
}

type ISession interface {
	CreateSession(form request.Session) (*model.Session, error)
	UpdateSessionExpireById(userRefId string, expireTime time.Time) (*model.Session, error)
	RemoveSessionById(userRefId string) (*model.Session, error)
	GetSessionById(userRefId string) (*model.Session, error)
}

func NewSessionEntity(resource *db.Resource) ISession {
	sessionRepo := resource.UmDb.Collection("sessions")
	var entity ISession = &sessionEntity{sessionRepo: sessionRepo}
	return entity
}

func (entity *sessionEntity) CreateSession(form request.Session) (*model.Session, error) {
	logrus.Info("CreateSession")
	ctx, cancel := utils.InitContext()
	defer cancel()

	var userRefId = primitive.NewObjectID()
	reference := model.Session{
		Id:          userRefId,
		UserId:      form.UserId,
		Type:        form.Type,
		Objective:   form.Objective,
		System:      form.System,
		ClientId:    form.ClientId,
		CreatedDate: time.Now(),
		ExpireDate:  form.ExpireDate,
	}
	_, err := entity.sessionRepo.InsertOne(ctx, reference)
	if err != nil {
		return nil, err
	}
	return &reference, nil
}

func (entity *sessionEntity) UpdateSessionExpireById(userRefId string, expireTime time.Time) (*model.Session, error) {
	logrus.Info("UpdateSessionExpireById")
	ctx, cancel := utils.InitContext()
	defer cancel()
	var reference model.Session
	objId, _ := primitive.ObjectIDFromHex(userRefId)
	err := entity.sessionRepo.FindOne(ctx, bson.M{"_id": objId}).Decode(&reference)
	if err != nil {
		return nil, err
	}
	reference.ExpireDate = expireTime
	isReturnNewDoc := options.After
	opts := &options.FindOneAndUpdateOptions{
		ReturnDocument: &isReturnNewDoc,
	}
	err = entity.sessionRepo.FindOneAndUpdate(ctx, bson.M{"_id": objId}, bson.M{"$set": reference}, opts).Decode(&reference)
	if err != nil {
		return nil, err
	}
	return &reference, nil
}

func (entity *sessionEntity) GetSessionById(userRefId string) (*model.Session, error) {
	logrus.Info("GetSessionById")
	ctx, cancel := utils.InitContext()
	defer cancel()
	var reference model.Session
	objId, _ := primitive.ObjectIDFromHex(userRefId)
	err := entity.sessionRepo.FindOne(ctx, bson.M{"_id": objId}).Decode(&reference)
	if err != nil {
		return nil, err
	}
	return &reference, nil
}

func (entity *sessionEntity) RemoveSessionById(userRefId string) (*model.Session, error) {
	logrus.Info("RemoveSessionById")
	ctx, cancel := utils.InitContext()
	defer cancel()
	var reference model.Session
	objId, _ := primitive.ObjectIDFromHex(userRefId)
	err := entity.sessionRepo.FindOne(ctx, bson.M{"_id": objId}).Decode(&reference)
	if err != nil {
		return nil, err
	}
	_, err = entity.sessionRepo.DeleteOne(ctx, bson.M{"_id": objId})
	if err != nil {
		return nil, err
	}
	return &reference, nil
}
