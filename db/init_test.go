package db

import (
	"context"
	"testing"

	"github.com/go-redis/redis/v8"
)

func TestRedisOptionsSupportsAddr(t *testing.T) {
	opts, err := redisOptions("localhost:6379")
	if err != nil {
		t.Fatalf("redisOptions returned error: %v", err)
	}
	if opts.Addr != "localhost:6379" {
		t.Fatalf("expected addr to be preserved, got %q", opts.Addr)
	}
}

func TestRedisOptionsSupportsURL(t *testing.T) {
	opts, err := redisOptions("redis://localhost:6379/2")
	if err != nil {
		t.Fatalf("redisOptions returned error: %v", err)
	}
	if opts.Addr != "localhost:6379" {
		t.Fatalf("expected parsed addr, got %q", opts.Addr)
	}
	if opts.DB != 2 {
		t.Fatalf("expected db 2, got %d", opts.DB)
	}
}

func TestRedisOptionsRejectsEmptyValue(t *testing.T) {
	if _, err := redisOptions("   "); err == nil {
		t.Fatal("expected error for empty REDIS_HOST")
	}
}

func TestMongoURIAddsDefaultScheme(t *testing.T) {
	got := mongoURI("localhost:27017")
	if got != "mongodb://localhost:27017" {
		t.Fatalf("expected mongodb scheme to be added, got %q", got)
	}
}

func TestMongoURIPreservesMongoScheme(t *testing.T) {
	got := mongoURI("mongodb+srv://cluster0.example.mongodb.net")
	if got != "mongodb+srv://cluster0.example.mongodb.net" {
		t.Fatalf("expected mongo uri to be preserved, got %q", got)
	}
}

func TestResourceCloseClosesRedisClient(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	resource := &Resource{RdDB: rdb}

	resource.Close()

	if err := rdb.Ping(context.Background()).Err(); err == nil {
		t.Fatal("expected redis client to be closed")
	}
}
