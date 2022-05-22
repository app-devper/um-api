package repository

import (
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strings"
	"time"
	"um/app/core/constant"
	"um/app/core/utils"
	"um/app/domain/model"
	"um/app/featues/request"
	"um/db"
)

type userEntity struct {
	userRepo *mongo.Collection
}

type IUser interface {
	CreateIndex() (string, error)
	GetUsers() ([]model.User, error)
	GetUserAll(clientId string) ([]model.User, error)
	GetUserByUsername(username string) (*model.User, error)
	GetUserById(id string) (*model.User, error)
	GetUserByClientId(id string, clientId string) (*model.User, error)
	CreateUser(form request.User, role string) (*model.User, error)
	RemoveUserById(id string) (*model.User, error)
	UpdateUserById(id string, form request.UpdateUser) (*model.User, error)
	UpdateStatusById(id string, form request.UpdateStatus) (*model.User, error)
	UpdateRoleById(id string, form request.UpdateRole) (*model.User, error)
	ChangePassword(id string, form request.ChangePassword) (*model.User, error)
	SetPassword(id string, form request.SetPassword) (*model.User, error)
}

func NewUserEntity(resource *db.Resource) IUser {
	userRepo := resource.UmDb.Collection("users")
	var entity IUser = &userEntity{userRepo: userRepo}
	_, _ = entity.CreateIndex()
	return entity
}

func (entity *userEntity) CreateIndex() (string, error) {
	ctx, cancel := utils.InitContext()
	defer cancel()
	mod := mongo.IndexModel{
		Keys: bson.M{
			"username": 1,
		},
		Options: options.Index().SetUnique(true),
	}
	ind, err := entity.userRepo.Indexes().CreateOne(ctx, mod)
	return ind, err
}

func (entity *userEntity) GetUsers() ([]model.User, error) {
	logrus.Info("GetUsers")
	var usersList []model.User
	ctx, cancel := utils.InitContext()
	defer cancel()
	cursor, err := entity.userRepo.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		var user model.User
		err = cursor.Decode(&user)
		if err != nil {
			logrus.Error(err)
			logrus.Info(cursor.Current)
		} else {
			usersList = append(usersList, user)
		}
	}
	if usersList == nil {
		usersList = []model.User{}
	}
	return usersList, nil
}

func (entity *userEntity) GetUserAll(clientId string) ([]model.User, error) {
	logrus.Info("GetUserAll")
	var usersList []model.User
	ctx, cancel := utils.InitContext()
	defer cancel()
	cursor, err := entity.userRepo.Find(ctx, bson.M{"clientId": clientId})
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		var user model.User
		err = cursor.Decode(&user)
		if err != nil {
			logrus.Error(err)
			logrus.Info(cursor.Current)
		} else {
			usersList = append(usersList, user)
		}
	}
	if usersList == nil {
		usersList = []model.User{}
	}
	return usersList, nil
}

func (entity *userEntity) GetUserByUsername(username string) (*model.User, error) {
	logrus.Info("GetUserByUsername")
	ctx, cancel := utils.InitContext()
	defer cancel()
	var user model.User
	err := entity.userRepo.FindOne(ctx, bson.M{"username": strings.TrimSpace(username)}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (entity *userEntity) CreateUser(form request.User, role string) (*model.User, error) {
	logrus.Info("CreateUser")
	ctx, cancel := utils.InitContext()
	defer cancel()

	var userId = primitive.NewObjectID()
	var createdBy = userId
	if form.CreatedBy != "" {
		createdBy, _ = primitive.ObjectIDFromHex(form.CreatedBy)
	}
	user := model.User{
		Id:          userId,
		FirstName:   form.FirstName,
		LastName:    form.LastName,
		Username:    form.Username,
		ClientId:    form.ClientId,
		Password:    utils.HashPassword(form.Password),
		Role:        role,
		Status:      constant.ACTIVE,
		CreatedBy:   createdBy,
		CreatedDate: time.Now(),
		UpdatedBy:   createdBy,
		UpdatedDate: time.Now(),
	}
	_, err := entity.userRepo.InsertOne(ctx, user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (entity *userEntity) GetUserById(id string) (*model.User, error) {
	logrus.Info("GetUserById")
	ctx, cancel := utils.InitContext()
	defer cancel()
	var user model.User
	objId, _ := primitive.ObjectIDFromHex(id)
	err := entity.userRepo.FindOne(ctx, bson.M{"_id": objId}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (entity *userEntity) GetUserByClientId(id string, clientId string) (*model.User, error) {
	logrus.Info("GetUserByClientId")
	ctx, cancel := utils.InitContext()
	defer cancel()
	var user model.User
	objId, _ := primitive.ObjectIDFromHex(id)
	err := entity.userRepo.FindOne(ctx, bson.M{"_id": objId, "clientId": clientId}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (entity *userEntity) RemoveUserById(id string) (*model.User, error) {
	logrus.Info("RemoveUserById")
	ctx, cancel := utils.InitContext()
	defer cancel()
	var user model.User
	objId, _ := primitive.ObjectIDFromHex(id)
	err := entity.userRepo.FindOne(ctx, bson.M{"_id": objId}).Decode(&user)
	if err != nil {
		return nil, err
	}
	_, err = entity.userRepo.DeleteOne(ctx, bson.M{"_id": objId})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (entity *userEntity) UpdateUserById(id string, form request.UpdateUser) (*model.User, error) {
	logrus.Info("UpdateUserById")
	ctx, cancel := utils.InitContext()
	defer cancel()
	objId, _ := primitive.ObjectIDFromHex(id)
	user, err := entity.GetUserById(id)
	if err != nil {
		return nil, err
	}

	user.FirstName = form.FirstName
	user.LastName = form.LastName
	user.Email = form.Email
	user.Phone = form.Phone
	user.UpdatedBy, _ = primitive.ObjectIDFromHex(form.UpdatedBy)
	user.UpdatedDate = time.Now()

	isReturnNewDoc := options.After
	opts := &options.FindOneAndUpdateOptions{
		ReturnDocument: &isReturnNewDoc,
	}
	err = entity.userRepo.FindOneAndUpdate(ctx, bson.M{"_id": objId}, bson.M{"$set": user}, opts).Decode(&user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (entity *userEntity) UpdateStatusById(id string, form request.UpdateStatus) (*model.User, error) {
	logrus.Info("UpdateStatusById")
	ctx, cancel := utils.InitContext()
	defer cancel()
	objId, _ := primitive.ObjectIDFromHex(id)
	user, err := entity.GetUserById(id)
	if err != nil {
		return nil, err
	}
	user.Status = form.Status
	user.UpdatedBy, _ = primitive.ObjectIDFromHex(form.UpdatedBy)
	user.UpdatedDate = time.Now()

	isReturnNewDoc := options.After
	opts := &options.FindOneAndUpdateOptions{
		ReturnDocument: &isReturnNewDoc,
	}
	err = entity.userRepo.FindOneAndUpdate(ctx, bson.M{"_id": objId}, bson.M{"$set": user}, opts).Decode(&user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (entity *userEntity) UpdateRoleById(id string, form request.UpdateRole) (*model.User, error) {
	logrus.Info("UpdateRoleById")
	ctx, cancel := utils.InitContext()
	defer cancel()
	objId, _ := primitive.ObjectIDFromHex(id)
	user, err := entity.GetUserById(id)
	if err != nil {
		return nil, err
	}

	user.Role = form.Role
	user.UpdatedBy, _ = primitive.ObjectIDFromHex(form.UpdatedBy)
	user.UpdatedDate = time.Now()

	isReturnNewDoc := options.After
	opts := &options.FindOneAndUpdateOptions{
		ReturnDocument: &isReturnNewDoc,
	}
	err = entity.userRepo.FindOneAndUpdate(ctx, bson.M{"_id": objId}, bson.M{"$set": user}, opts).Decode(&user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (entity *userEntity) ChangePassword(id string, form request.ChangePassword) (*model.User, error) {
	logrus.Info("ChangePassword")
	ctx, cancel := utils.InitContext()
	defer cancel()
	objId, _ := primitive.ObjectIDFromHex(id)
	user, err := entity.GetUserById(id)
	if err != nil {
		return nil, err
	}
	user.Password = utils.HashPassword(form.NewPassword)
	user.UpdatedBy = objId
	user.UpdatedDate = time.Now()
	isReturnNewDoc := options.After
	opts := &options.FindOneAndUpdateOptions{
		ReturnDocument: &isReturnNewDoc,
	}
	err = entity.userRepo.FindOneAndUpdate(ctx, bson.M{"_id": objId}, bson.M{"$set": user}, opts).Decode(&user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (entity *userEntity) SetPassword(id string, form request.SetPassword) (*model.User, error) {
	logrus.Info("SetPassword")
	ctx, cancel := utils.InitContext()
	defer cancel()
	objId, _ := primitive.ObjectIDFromHex(id)
	user, err := entity.GetUserById(id)
	if err != nil {
		return nil, err
	}
	user.Password = utils.HashPassword(form.Password)
	user.UpdatedBy = objId
	user.UpdatedDate = time.Now()
	isReturnNewDoc := options.After
	opts := &options.FindOneAndUpdateOptions{
		ReturnDocument: &isReturnNewDoc,
	}
	err = entity.userRepo.FindOneAndUpdate(ctx, bson.M{"_id": objId}, bson.M{"$set": user}, opts).Decode(&user)
	if err != nil {
		return nil, err
	}
	return user, nil
}
