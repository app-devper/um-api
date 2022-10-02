package repository

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"um/config"
	"um/db"
)

type sessionEntity struct {
	rdb *redis.Client
}

type ISession interface {
	CreateSession(userId string) (string, error)
	UpdateSessionExpireById(sessionId string) error
	RemoveSessionById(sessionId string) error
	GetSessionById(sessionId string) (string, error)
}

func NewSessionEntity(resource *db.Resource) ISession {
	var entity ISession = &sessionEntity{rdb: resource.RdDB}
	return entity
}

func (entity *sessionEntity) CreateSession(userId string) (string, error) {
	logrus.Info("CreateSession")
	id := uuid.New()
	sessionId := id.String()
	err := entity.rdb.Set(context.Background(), sessionId, userId, config.AccessTokenTime).Err()
	if err != nil {
		return "", err
	}
	return sessionId, nil
}

func (entity *sessionEntity) UpdateSessionExpireById(sessionId string) error {
	logrus.Info("UpdateSessionExpireById")
	err := entity.rdb.Expire(context.Background(), sessionId, config.AccessTokenTime).Err()
	if err != nil {
		return err
	}
	return err
}

func (entity *sessionEntity) GetSessionById(sessionId string) (string, error) {
	logrus.Info("GetSessionById")
	result, err := entity.rdb.Get(context.Background(), sessionId).Result()
	if err != nil {
		return "", err
	}
	return result, nil
}

func (entity *sessionEntity) RemoveSessionById(sessionId string) error {
	logrus.Info("RemoveSessionById")
	_, err := entity.rdb.Del(context.Background(), sessionId).Result()
	if err != nil {
		return err
	}
	return nil
}
