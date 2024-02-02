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

type systemEntity struct {
	systemRepo *mongo.Collection
}

type ISystem interface {
	GetSystems(form request.GetSystems) ([]model.System, error)
	GetSystemsByClientId(clientId string) ([]model.System, error)
	GetSystemsByCode(systemCode string) ([]model.System, error)
	GetSystemById(id string) (*model.System, error)
	GetSystem(clientId string, systemCode string) (*model.System, error)
	CreateSystem(form request.System) (*model.System, error)
	RemoveSystemById(id string) (*model.System, error)
	UpdateSystemById(id string, form request.UpdateSystem) (*model.System, error)
}

func NewSystemEntity(resource *db.Resource) ISystem {
	systemRepo := resource.UmDb.Collection("systems")
	var entity ISystem = &systemEntity{systemRepo: systemRepo}
	return entity
}

func (entity systemEntity) GetSystems(form request.GetSystems) (items []model.System, err error) {
	logrus.Info("GetSystems")
	ctx, cancel := utils.InitContext()
	defer cancel()
	var queries = bson.M{}
	if form.SystemCode != "" {
		queries["systemCode"] = form.SystemCode
	}
	if form.ClientId != "" {
		queries["clientId"] = form.ClientId
	}
	cursor, err := entity.systemRepo.Find(ctx, queries)
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		var item model.System
		err = cursor.Decode(&item)
		if err != nil {
			logrus.Error(err)
			logrus.Info(cursor.Current)
		} else {
			items = append(items, item)
		}
	}
	if items == nil {
		items = []model.System{}
	}
	return items, nil
}

func (entity systemEntity) GetSystem(clientId string, systemCode string) (*model.System, error) {
	logrus.Info("GetSystem")
	ctx, cancel := utils.InitContext()
	defer cancel()
	var item model.System
	err := entity.systemRepo.FindOne(ctx, bson.M{"clientId": clientId, "systemCode": systemCode}).Decode(&item)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (entity systemEntity) GetSystemsByClientId(clientId string) (items []model.System, err error) {
	logrus.Info("GetSystemsByClientId")
	ctx, cancel := utils.InitContext()
	defer cancel()
	cursor, err := entity.systemRepo.Find(ctx, bson.M{"clientId": clientId})
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		var item model.System
		err = cursor.Decode(&item)
		if err != nil {
			logrus.Error(err)
			logrus.Info(cursor.Current)
		} else {
			items = append(items, item)
		}
	}
	if items == nil {
		items = []model.System{}
	}
	return items, nil
}

func (entity systemEntity) GetSystemsByCode(systemCode string) (items []model.System, err error) {
	logrus.Info("GetSystemsByCode")
	ctx, cancel := utils.InitContext()
	defer cancel()
	cursor, err := entity.systemRepo.Find(ctx, bson.M{"systemCode": systemCode})
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		var item model.System
		err = cursor.Decode(&item)
		if err != nil {
			logrus.Error(err)
			logrus.Info(cursor.Current)
		} else {
			items = append(items, item)
		}
	}
	if items == nil {
		items = []model.System{}
	}
	return items, nil
}

func (entity systemEntity) GetSystemById(id string) (*model.System, error) {
	logrus.Info("GetSystemById")
	ctx, cancel := utils.InitContext()
	defer cancel()
	var item model.System
	err := entity.systemRepo.FindOne(ctx, bson.M{"_id": id}).Decode(&item)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (entity systemEntity) CreateSystem(form request.System) (*model.System, error) {
	logrus.Info("CreateSystem")
	ctx, cancel := utils.InitContext()
	defer cancel()

	var id = primitive.NewObjectID()
	createdBy, _ := primitive.ObjectIDFromHex(form.CreatedBy)
	item := model.System{
		Id:          id,
		SystemName:  form.SystemName,
		SystemCode:  form.SystemCode,
		Host:        form.Host,
		ClientId:    form.ClientId,
		CreatedBy:   createdBy,
		CreatedDate: time.Now(),
		UpdatedBy:   createdBy,
		UpdatedDate: time.Now(),
	}
	_, err := entity.systemRepo.InsertOne(ctx, item)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (entity systemEntity) RemoveSystemById(id string) (*model.System, error) {
	logrus.Info("RemoveSystemById")
	ctx, cancel := utils.InitContext()
	defer cancel()
	var item model.System
	objId, _ := primitive.ObjectIDFromHex(id)
	err := entity.systemRepo.FindOne(ctx, bson.M{"_id": objId}).Decode(&item)
	if err != nil {
		return nil, err
	}
	_, err = entity.systemRepo.DeleteOne(ctx, bson.M{"_id": objId})
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (entity systemEntity) UpdateSystemById(id string, form request.UpdateSystem) (*model.System, error) {
	logrus.Info("UpdateSystemById")
	ctx, cancel := utils.InitContext()
	defer cancel()
	objId, _ := primitive.ObjectIDFromHex(id)
	item, err := entity.GetSystemById(id)
	if err != nil {
		return nil, err
	}

	item.SystemName = form.SystemName
	item.Host = form.Host
	item.UpdatedBy, _ = primitive.ObjectIDFromHex(form.UpdatedBy)
	item.UpdatedDate = time.Now()

	isReturnNewDoc := options.After
	opts := &options.FindOneAndUpdateOptions{
		ReturnDocument: &isReturnNewDoc,
	}
	err = entity.systemRepo.FindOneAndUpdate(ctx, bson.M{"_id": objId}, bson.M{"$set": item}, opts).Decode(&item)
	if err != nil {
		return nil, err
	}
	return item, nil
}
