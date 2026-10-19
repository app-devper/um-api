package repository

import (
	"context"
	"errors"
	"time"
	"um/app/domain/model"
	"um/db"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ErrPasskeyNotFound: no Passkey with that ID for that User.
var ErrPasskeyNotFound = errors.New("passkey not found")

// IPasskey stores Users' Passkeys, keyed by credential ID.
type IPasskey interface {
	Create(p model.Passkey) error
	Get(id string) (*model.Passkey, error)
	ListByUser(userId string) ([]model.Passkey, error)
	RecordUse(id string, credential []byte, at time.Time) error
	Delete(id, userId string) error
	DeleteAllForUser(userId string) error
}

type passkeyEntity struct {
	collection *mongo.Collection
}

func NewPasskeyEntity(resource *db.Resource) IPasskey {
	collection := resource.UmDb.Collection("passkeys")
	_, _ = collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{Keys: bson.D{{Key: "userId", Value: 1}}})
	return &passkeyEntity{collection: collection}
}

func (e *passkeyEntity) Create(p model.Passkey) error {
	_, err := e.collection.InsertOne(context.Background(), p)
	return err
}

func (e *passkeyEntity) Get(id string) (*model.Passkey, error) {
	p := model.Passkey{}
	err := e.collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrPasskeyNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (e *passkeyEntity) ListByUser(userId string) ([]model.Passkey, error) {
	cursor, err := e.collection.Find(context.Background(), bson.M{"userId": userId}, options.Find().SetSort(bson.D{{Key: "createdDate", Value: 1}}))
	if err != nil {
		return nil, err
	}
	keys := []model.Passkey{}
	if err := cursor.All(context.Background(), &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

func (e *passkeyEntity) RecordUse(id string, credential []byte, at time.Time) error {
	result, err := e.collection.UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{"$set": bson.M{"credential": credential, "lastUsedDate": at}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrPasskeyNotFound
	}
	return nil
}

func (e *passkeyEntity) Delete(id, userId string) error {
	result, err := e.collection.DeleteOne(context.Background(), bson.M{"_id": id, "userId": userId})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrPasskeyNotFound
	}
	return nil
}

func (e *passkeyEntity) DeleteAllForUser(userId string) error {
	_, err := e.collection.DeleteMany(context.Background(), bson.M{"userId": userId})
	return err
}

const (
	ceremonyPrefix = "passkey_ceremony:"
	// CeremonyTTL is how long a Passkey challenge can be answered.
	CeremonyTTL = 5 * time.Minute
)

// ErrCeremonyNotFound: the Ceremony expired, was already answered, or never existed.
var ErrCeremonyNotFound = errors.New("passkey ceremony not found or expired")

// IPasskeyCeremony keeps a WebAuthn challenge until it is answered once.
type IPasskeyCeremony interface {
	Save(kind string, state []byte) (string, error)
	Consume(kind, id string) ([]byte, error)
}

type ceremonyEntity struct {
	rdb *redis.Client
}

func NewPasskeyCeremonyEntity(resource *db.Resource) IPasskeyCeremony {
	return &ceremonyEntity{rdb: resource.RdDB}
}

func (e *ceremonyEntity) Save(kind string, state []byte) (string, error) {
	id := uuid.New().String()
	if err := e.rdb.Set(context.Background(), ceremonyPrefix+kind+":"+id, state, CeremonyTTL).Err(); err != nil {
		return "", err
	}
	return id, nil
}

func (e *ceremonyEntity) Consume(kind, id string) ([]byte, error) {
	state, err := e.rdb.GetDel(context.Background(), ceremonyPrefix+kind+":"+id).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrCeremonyNotFound
	}
	return state, err
}
