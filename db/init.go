package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"um/app/core/config"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Resource struct {
	UmDb        *mongo.Database
	RdDB        *redis.Client
	mongoClient *mongo.Client
}

// Close use this method to close database connection
func (r *Resource) Close() {
	logrus.Warning("Closing all db connections")
	if r == nil {
		return
	}
	if r.mongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := r.mongoClient.Disconnect(ctx); err != nil {
			logrus.Error(err)
		}
	}
	if r.RdDB != nil {
		if err := r.RdDB.Close(); err != nil && !errors.Is(err, redis.Nil) {
			logrus.Error(err)
		}
	}
}

func InitResource(cfg *config.AppConfig) (*Resource, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI(cfg.MongoHost)))
	if err != nil {
		return nil, err
	}
	if err := mongoClient.Ping(ctx, nil); err != nil {
		return nil, err
	}

	// Redis client
	redisOp, err := redisOptions(cfg.RedisHost)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(redisOp)
	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}

	return &Resource{
		UmDb:        mongoClient.Database(cfg.MongoUMDBName),
		RdDB:        rdb,
		mongoClient: mongoClient,
	}, nil
}

func redisOptions(redisHost string) (*redis.Options, error) {
	redisHost = strings.TrimSpace(redisHost)
	if redisHost == "" {
		return nil, fmt.Errorf("REDIS_HOST is required")
	}
	if strings.Contains(redisHost, "://") {
		return redis.ParseURL(redisHost)
	}
	return &redis.Options{Addr: redisHost}, nil
}

func mongoURI(mongoHost string) string {
	mongoHost = strings.TrimSpace(mongoHost)
	if strings.HasPrefix(mongoHost, "mongodb://") || strings.HasPrefix(mongoHost, "mongodb+srv://") {
		return mongoHost
	}
	return "mongodb://" + mongoHost
}
