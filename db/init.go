package db

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"os"
	"time"
)

type Resource struct {
	UmDb *mongo.Database
	RdDB *redis.Client
}

// Close use this method to close database connection
func (r *Resource) Close() {
	logrus.Warning("Closing all db connections")
}

func InitResource() (*Resource, error) {
	err := godotenv.Load(".env")
	if err != nil {
		log.Print(err)
	}
	host := os.Getenv("MONGO_HOST")
	umDbName := os.Getenv("MONGO_UM_DB_NAME")
	mongoClient, err := mongo.NewClient(options.Client().ApplyURI(host))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = mongoClient.Connect(ctx)
	if err != nil {
		return nil, err
	}
	redisHost := os.Getenv("REDIS_HOST")
	redisOp, err := redis.ParseURL(redisHost)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(redisOp)
	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}
	return &Resource{
		UmDb: mongoClient.Database(umDbName),
		RdDB: rdb,
	}, nil
}
