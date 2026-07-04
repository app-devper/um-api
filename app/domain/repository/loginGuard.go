package repository

import (
	"context"
	"time"
	"um/db"

	"github.com/go-redis/redis/v8"
)

const (
	loginFailPrefix = "login_fail:"
	loginLockPrefix = "login_lock:"
	LoginFailWindow = 15 * time.Minute
	LoginLockDur    = 15 * time.Minute
	LoginMaxFails   = 5
)

type loginGuardEntity struct {
	rdb     *redis.Client
	enabled bool
}

type ILoginGuard interface {
	IsLocked(username string) (bool, error)
	RecordFailure(username string) error
	Reset(username string) error
	Unlock(username string) error
}

func NewLoginGuardEntity(resource *db.Resource, enabled bool) ILoginGuard {
	return &loginGuardEntity{rdb: resource.RdDB, enabled: enabled}
}

func (e *loginGuardEntity) IsLocked(username string) (bool, error) {
	if !e.enabled {
		return false, nil
	}
	ctx := context.Background()
	n, err := e.rdb.Exists(ctx, loginLockPrefix+username).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (e *loginGuardEntity) RecordFailure(username string) error {
	if !e.enabled {
		return nil
	}
	ctx := context.Background()
	failKey := loginFailPrefix + username
	count, err := e.rdb.Incr(ctx, failKey).Result()
	if err != nil {
		return err
	}
	if count == 1 {
		if err := e.rdb.Expire(ctx, failKey, LoginFailWindow).Err(); err != nil {
			return err
		}
	}
	if count >= LoginMaxFails {
		if err := e.rdb.Set(ctx, loginLockPrefix+username, "1", LoginLockDur).Err(); err != nil {
			return err
		}
	}
	return nil
}

func (e *loginGuardEntity) Reset(username string) error {
	if !e.enabled {
		return nil
	}
	ctx := context.Background()
	_, err := e.rdb.Del(ctx, loginFailPrefix+username, loginLockPrefix+username).Result()
	return err
}

// Unlock always runs regardless of enabled flag so admins can clear
// leftover state after toggling the feature off.
func (e *loginGuardEntity) Unlock(username string) error {
	ctx := context.Background()
	_, err := e.rdb.Del(ctx, loginFailPrefix+username, loginLockPrefix+username).Result()
	return err
}
