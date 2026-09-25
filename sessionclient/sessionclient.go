// Package sessionclient lets another service confirm that the UM session
// behind an access token it has already verified is still live, by reading
// UM's session store directly (um-api ADR-0003).
//
// UM revokes every session of a user whose role, status, password, or account
// changes, so while a session exists its token's role and client claims are
// current. Answers are cached per session for at most DefaultTTL, so a
// revocation in UM takes effect within about 30 seconds.
//
// UM owns the key layout; its tests read what UM writes through this package.
package sessionclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// KeyPrefix is the Redis key prefix UM stores sessions under: "session:<jti>".
const KeyPrefix = "session:"

// DefaultTTL bounds how long a confirmed session is trusted without asking
// the store again.
const DefaultTTL = 30 * time.Second

// lookupTimeout bounds one store lookup, including dial, so an outage turns
// into a prompt ErrUnavailable instead of a hung request.
const lookupTimeout = time.Second

var (
	// ErrSessionRejected means the session is gone (logged out, revoked, or
	// expired) or was issued for another system. The caller must refuse.
	ErrSessionRejected = errors.New("session rejected")
	// ErrUnavailable means the store could not answer.
	ErrUnavailable = errors.New("UM session store unavailable")
)

// Session is the part of UM's stored session a caller needs.
type Session struct {
	UserId string `json:"userId"`
	System string `json:"system"`
}

// Store looks up a session. It returns ErrSessionRejected when the session
// does not exist and ErrUnavailable when it cannot tell.
type Store interface {
	Session(ctx context.Context, sessionID string) (Session, error)
}

// RedisStore reads UM's session keys. It never writes.
type RedisStore struct {
	rdb *redis.Client
}

func NewRedisStore(rdb *redis.Client) *RedisStore {
	return &RedisStore{rdb: rdb}
}

// NewRedisClient connects to UM's Redis from a host:port or redis:// URL with
// timeouts suited to a per-request lookup: one second, no retries.
func NewRedisClient(hostOrURL string) (*redis.Client, error) {
	opts := &redis.Options{Addr: hostOrURL}
	if strings.Contains(hostOrURL, "://") {
		parsed, err := redis.ParseURL(hostOrURL)
		if err != nil {
			return nil, err
		}
		opts = parsed
	}
	// go-redis ignores context deadlines on sockets unless told otherwise and
	// retries with its own 3-second timeouts.
	opts.ContextTimeoutEnabled = true
	opts.DialTimeout = lookupTimeout
	opts.ReadTimeout = lookupTimeout
	opts.WriteTimeout = lookupTimeout
	opts.MaxRetries = -1
	return redis.NewClient(opts), nil
}

func (s *RedisStore) Session(ctx context.Context, sessionID string) (Session, error) {
	ctx, cancel := context.WithTimeout(ctx, lookupTimeout)
	defer cancel()
	raw, err := s.rdb.Get(ctx, KeyPrefix+sessionID).Result()
	if errors.Is(err, redis.Nil) {
		return Session{}, ErrSessionRejected
	}
	if err != nil {
		return Session{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	var session Session
	if err := json.Unmarshal([]byte(raw), &session); err != nil || session.UserId == "" {
		return Session{}, fmt.Errorf("%w: unreadable session", ErrUnavailable)
	}
	return session, nil
}

// Checker confirms sessions with a per-session cache. A nil *Checker, or one
// built without a store, is disabled: Check always succeeds.
type Checker struct {
	store Store
	ttl   time.Duration
	now   func() time.Time

	mu      sync.Mutex
	entries map[string]time.Time
}

func NewChecker(store Store) *Checker {
	return &Checker{store: store, ttl: DefaultTTL, now: time.Now, entries: map[string]time.Time{}}
}

// New builds a Checker for UM's Redis at hostOrURL. An empty hostOrURL gives
// a disabled Checker, so a service can roll out before it is configured.
func New(hostOrURL string) (*Checker, error) {
	if hostOrURL == "" {
		return nil, nil
	}
	rdb, err := NewRedisClient(hostOrURL)
	if err != nil {
		return nil, err
	}
	return NewChecker(NewRedisStore(rdb)), nil
}

// Enabled reports whether Check consults a store.
func (c *Checker) Enabled() bool {
	return c != nil && c.store != nil
}

// Check confirms that sessionID, taken from a token the caller has already
// verified, is live in UM and was issued for system. It returns nil,
// ErrSessionRejected, or ErrUnavailable (possibly wrapped).
func (c *Checker) Check(ctx context.Context, sessionID, system string) error {
	if !c.Enabled() {
		return nil
	}
	if sessionID == "" {
		return ErrSessionRejected
	}
	now := c.now()
	c.mu.Lock()
	checkedAt, ok := c.entries[sessionID]
	c.mu.Unlock()
	if ok && now.Sub(checkedAt) < c.ttl {
		return nil
	}

	session, err := c.store.Session(ctx, sessionID)
	// Sessions created before UM recorded the system carry none; accept them.
	if err == nil && session.System != "" && session.System != system {
		err = fmt.Errorf("%w: session was issued for system %q", ErrSessionRejected, session.System)
	}
	if err != nil {
		if !errors.Is(err, ErrUnavailable) {
			c.forget(sessionID)
		}
		return err
	}

	c.mu.Lock()
	c.entries[sessionID] = now
	if len(c.entries) > 10000 {
		for id, at := range c.entries {
			if now.Sub(at) >= c.ttl {
				delete(c.entries, id)
			}
		}
	}
	c.mu.Unlock()
	return nil
}

func (c *Checker) forget(sessionID string) {
	c.mu.Lock()
	delete(c.entries, sessionID)
	c.mu.Unlock()
}

// ReadOnlyDuringOutage is the default outage policy: while the store is
// unavailable, safe methods may continue under the signed token and writes
// are refused.
func ReadOnlyDuringOutage(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}
