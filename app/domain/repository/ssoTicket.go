package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"
	"um/db"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

const (
	ssoTicketPrefix = "sso_ticket:"
	SSOTicketTTL    = 60 * time.Second
)

var ErrTicketNotFound = errors.New("sso ticket not found or expired")

type TicketPayload struct {
	UserId   string `json:"userId"`
	Role     string `json:"role"`
	ClientId string `json:"clientId"`
	System   string `json:"system"`
}

type ssoTicketEntity struct {
	rdb *redis.Client
}

type ISSOTicket interface {
	Create(payload TicketPayload) (string, error)
	Consume(ticket string) (*TicketPayload, error)
}

func NewSSOTicketEntity(resource *db.Resource) ISSOTicket {
	return &ssoTicketEntity{rdb: resource.RdDB}
}

func (e *ssoTicketEntity) Create(payload TicketPayload) (string, error) {
	ctx := context.Background()
	ticket := uuid.New().String()
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	if err := e.rdb.Set(ctx, ssoTicketPrefix+ticket, body, SSOTicketTTL).Err(); err != nil {
		return "", err
	}
	return ticket, nil
}

func (e *ssoTicketEntity) Consume(ticket string) (*TicketPayload, error) {
	ctx := context.Background()
	body, err := e.rdb.GetDel(ctx, ssoTicketPrefix+ticket).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrTicketNotFound
		}
		return nil, err
	}
	payload := TicketPayload{}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}
