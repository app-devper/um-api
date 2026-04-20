package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"
	"um/db"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	sessionPrefix      = "session:"
	userSessionsPrefix = "user_sessions:"
)

type SessionMetadata struct {
	UserAgent string
	IPAddress string
	System    string
}

type SessionData struct {
	UserId       string    `json:"userId"`
	CreatedAt    time.Time `json:"createdAt"`
	LastActivity time.Time `json:"lastActivity"`
	UserAgent    string    `json:"userAgent"`
	IPAddress    string    `json:"ipAddress"`
	System       string    `json:"system"`
}

type SessionInfo struct {
	SessionId    string    `json:"sessionId"`
	CreatedAt    time.Time `json:"createdAt"`
	LastActivity time.Time `json:"lastActivity"`
	UserAgent    string    `json:"userAgent"`
	IPAddress    string    `json:"ipAddress"`
	System       string    `json:"system"`
	Current      bool      `json:"current"`
}

type sessionEntity struct {
	rdb *redis.Client
}

type ISession interface {
	CreateSession(userId string, expiration time.Duration, metadata SessionMetadata) (string, error)
	UpdateSessionExpireById(sessionId string, expiration time.Duration) error
	RemoveSessionById(sessionId string) error
	GetSessionById(sessionId string) (string, error)
	ListUserSessions(userId string, currentSessionId string) ([]SessionInfo, error)
	RevokeOtherSessions(userId string, currentSessionId string) (int, error)
}

func NewSessionEntity(resource *db.Resource) ISession {
	return &sessionEntity{rdb: resource.RdDB}
}

func (e *sessionEntity) CreateSession(userId string, expiration time.Duration, metadata SessionMetadata) (string, error) {
	logrus.Info("CreateSession")
	ctx := context.Background()
	sessionId := uuid.New().String()
	now := time.Now()
	data := SessionData{
		UserId:       userId,
		CreatedAt:    now,
		LastActivity: now,
		UserAgent:    metadata.UserAgent,
		IPAddress:    metadata.IPAddress,
		System:       metadata.System,
	}
	body, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	if err := e.rdb.Set(ctx, sessionPrefix+sessionId, body, expiration).Err(); err != nil {
		return "", err
	}
	if err := e.rdb.SAdd(ctx, userSessionsPrefix+userId, sessionId).Err(); err != nil {
		return "", err
	}
	e.rdb.Expire(ctx, userSessionsPrefix+userId, expiration)
	return sessionId, nil
}

func (e *sessionEntity) GetSessionById(sessionId string) (string, error) {
	logrus.Info("GetSessionById")
	data, err := e.loadSession(sessionId)
	if err != nil {
		return "", err
	}
	return data.UserId, nil
}

func (e *sessionEntity) UpdateSessionExpireById(sessionId string, expiration time.Duration) error {
	logrus.Info("UpdateSessionExpireById")
	ctx := context.Background()
	data, err := e.loadSession(sessionId)
	if err != nil {
		return err
	}
	data.LastActivity = time.Now()
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if err := e.rdb.Set(ctx, sessionPrefix+sessionId, body, expiration).Err(); err != nil {
		return err
	}
	e.rdb.Expire(ctx, userSessionsPrefix+data.UserId, expiration)
	return nil
}

func (e *sessionEntity) RemoveSessionById(sessionId string) error {
	logrus.Info("RemoveSessionById")
	ctx := context.Background()
	data, err := e.loadSession(sessionId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}
		return err
	}
	if _, err := e.rdb.Del(ctx, sessionPrefix+sessionId).Result(); err != nil {
		return err
	}
	e.rdb.SRem(ctx, userSessionsPrefix+data.UserId, sessionId)
	return nil
}

func (e *sessionEntity) ListUserSessions(userId string, currentSessionId string) ([]SessionInfo, error) {
	ctx := context.Background()
	ids, err := e.rdb.SMembers(ctx, userSessionsPrefix+userId).Result()
	if err != nil {
		return nil, err
	}
	items := make([]SessionInfo, 0, len(ids))
	for _, id := range ids {
		data, err := e.loadSession(id)
		if err != nil {
			if errors.Is(err, redis.Nil) {
				e.rdb.SRem(ctx, userSessionsPrefix+userId, id)
				continue
			}
			logrus.Warn("loadSession error: ", err)
			continue
		}
		items = append(items, SessionInfo{
			SessionId:    id,
			CreatedAt:    data.CreatedAt,
			LastActivity: data.LastActivity,
			UserAgent:    data.UserAgent,
			IPAddress:    data.IPAddress,
			System:       data.System,
			Current:      id == currentSessionId,
		})
	}
	return items, nil
}

func (e *sessionEntity) RevokeOtherSessions(userId string, currentSessionId string) (int, error) {
	ctx := context.Background()
	ids, err := e.rdb.SMembers(ctx, userSessionsPrefix+userId).Result()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, id := range ids {
		if id == currentSessionId {
			continue
		}
		if _, err := e.rdb.Del(ctx, sessionPrefix+id).Result(); err != nil {
			logrus.Warn("revoke session del error: ", err)
			continue
		}
		e.rdb.SRem(ctx, userSessionsPrefix+userId, id)
		count++
	}
	return count, nil
}

func (e *sessionEntity) loadSession(sessionId string) (*SessionData, error) {
	ctx := context.Background()
	raw, err := e.rdb.Get(ctx, sessionPrefix+sessionId).Result()
	if err != nil {
		return nil, err
	}
	data := SessionData{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, err
	}
	return &data, nil
}
