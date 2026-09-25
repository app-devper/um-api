package repository

import (
	"context"
	"errors"
	"testing"
	"time"
	"um/db"

	"github.com/app-devper/um-api/sessionclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func newRedis(t *testing.T) (*db.Resource, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return &db.Resource{RdDB: client}, mr
}

func TestSessionLifecycle(t *testing.T) {
	res, mr := newRedis(t)
	store := NewSessionEntity(res)

	first, err := store.CreateSession("u1", time.Hour, SessionMetadata{UserAgent: "ua", IPAddress: "1.2.3.4", System: "UM"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	second, _ := store.CreateSession("u1", time.Hour, SessionMetadata{System: "POS"})

	if data, err := store.GetSessionById(first); err != nil || data.UserId != "u1" || data.System != "UM" {
		t.Fatalf("expected u1 on UM, got %+v err=%v", data, err)
	}

	items, err := store.ListUserSessions("u1", first)
	if err != nil || len(items) != 2 {
		t.Fatalf("expected 2 sessions, got %d err=%v", len(items), err)
	}
	for _, item := range items {
		if item.Current != (item.SessionId == first) {
			t.Fatalf("wrong current flag on %+v", item)
		}
		if item.SessionId == first && (item.System != "UM" || item.IPAddress != "1.2.3.4") {
			t.Fatalf("metadata not stored: %+v", item)
		}
	}

	if err := store.RemoveSessionById(second); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := store.GetSessionById(second); !errors.Is(err, redis.Nil) {
		t.Fatalf("expected removed session gone, got %v", err)
	}
	if members, _ := mr.Members(userSessionsPrefix + "u1"); len(members) != 1 || members[0] != first {
		t.Fatalf("expected index to hold only %s, got %v", first, members)
	}
	if err := store.RemoveSessionById(second); err != nil {
		t.Fatalf("removing twice should be a no-op, got %v", err)
	}
}

func TestSessionExpiryAndRenewal(t *testing.T) {
	res, mr := newRedis(t)
	store := NewSessionEntity(res)
	id, _ := store.CreateSession("u1", time.Hour, SessionMetadata{})

	mr.FastForward(50 * time.Minute)
	if err := store.UpdateSessionExpireById(id, time.Hour); err != nil {
		t.Fatalf("renew: %v", err)
	}
	mr.FastForward(50 * time.Minute)
	if _, err := store.GetSessionById(id); err != nil {
		t.Fatalf("renewed session should survive, got %v", err)
	}
	if ttl := mr.TTL(userSessionsPrefix + "u1"); ttl <= 0 {
		t.Fatalf("expected user index TTL to be renewed, got %v", ttl)
	}

	mr.FastForward(time.Hour)
	if _, err := store.GetSessionById(id); !errors.Is(err, redis.Nil) {
		t.Fatalf("expected expired session, got %v", err)
	}
	if err := store.UpdateSessionExpireById(id, time.Hour); err == nil {
		t.Fatal("renewing an expired session must fail")
	}
}

func TestRevokeOtherSessions(t *testing.T) {
	res, _ := newRedis(t)
	store := NewSessionEntity(res)
	keep, _ := store.CreateSession("u1", time.Hour, SessionMetadata{})
	store.CreateSession("u1", time.Hour, SessionMetadata{})
	store.CreateSession("u1", time.Hour, SessionMetadata{})
	other, _ := store.CreateSession("u2", time.Hour, SessionMetadata{})

	count, err := store.RevokeOtherSessions("u1", keep)
	if err != nil || count != 2 {
		t.Fatalf("expected 2 revoked, got %d err=%v", count, err)
	}
	if _, err := store.GetSessionById(keep); err != nil {
		t.Fatalf("current session must survive, got %v", err)
	}
	if _, err := store.GetSessionById(other); err != nil {
		t.Fatalf("another user's session must survive, got %v", err)
	}

	count, _ = store.RevokeOtherSessions("u1", "")
	if count != 1 {
		t.Fatalf("empty current should revoke every session, got %d", count)
	}
	if items, _ := store.ListUserSessions("u1", ""); len(items) != 0 {
		t.Fatalf("expected no sessions left, got %+v", items)
	}
}

func TestListUserSessionsDropsExpiredIds(t *testing.T) {
	res, mr := newRedis(t)
	store := NewSessionEntity(res)
	short, _ := store.CreateSession("u1", time.Minute, SessionMetadata{})
	long, _ := store.CreateSession("u1", time.Hour, SessionMetadata{})

	mr.FastForward(2 * time.Minute)
	items, err := store.ListUserSessions("u1", "")
	if err != nil || len(items) != 1 || items[0].SessionId != long {
		t.Fatalf("expected only %s, got %+v err=%v", long, items, err)
	}
	if mr.Exists(userSessionsPrefix+"u1") && contains(mustMembers(t, mr, userSessionsPrefix+"u1"), short) {
		t.Fatal("expected expired id pruned from the user index")
	}
}

func TestSSOTicketIsSingleUse(t *testing.T) {
	res, mr := newRedis(t)
	tickets := NewSSOTicketEntity(res)

	ticket, err := tickets.Create(TicketPayload{UserId: "u1", System: "POS"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	payload, err := tickets.Consume(ticket)
	if err != nil || payload.UserId != "u1" || payload.System != "POS" {
		t.Fatalf("expected payload, got %+v err=%v", payload, err)
	}
	if _, err := tickets.Consume(ticket); !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("second consume must fail, got %v", err)
	}

	expiring, _ := tickets.Create(TicketPayload{UserId: "u1"})
	mr.FastForward(SSOTicketTTL + time.Second)
	if _, err := tickets.Consume(expiring); !errors.Is(err, ErrTicketNotFound) {
		t.Fatalf("expired ticket must fail, got %v", err)
	}
}

func TestLoginGuardLocksAfterMaxFailures(t *testing.T) {
	res, mr := newRedis(t)
	guard := NewLoginGuardEntity(res, true)

	for i := 0; i < LoginMaxFails-1; i++ {
		_ = guard.RecordFailure("alice")
	}
	if locked, _ := guard.IsLocked("alice"); locked {
		t.Fatal("must not lock before the limit")
	}
	_ = guard.RecordFailure("alice")
	if locked, _ := guard.IsLocked("alice"); !locked {
		t.Fatal("expected lock at the limit")
	}

	mr.FastForward(LoginLockDur + time.Second)
	if locked, _ := guard.IsLocked("alice"); locked {
		t.Fatal("lock should expire")
	}

	for i := 0; i < LoginMaxFails; i++ {
		_ = guard.RecordFailure("bob")
	}
	if err := guard.Unlock("bob"); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	if locked, _ := guard.IsLocked("bob"); locked {
		t.Fatal("unlock should clear the lock")
	}
}

func TestLoginGuardDisabledNeverLocks(t *testing.T) {
	res, _ := newRedis(t)
	guard := NewLoginGuardEntity(res, false)
	for i := 0; i < LoginMaxFails*2; i++ {
		_ = guard.RecordFailure("alice")
	}
	if locked, _ := guard.IsLocked("alice"); locked {
		t.Fatal("disabled guard must never lock")
	}
}

func mustMembers(t *testing.T, mr *miniredis.Miniredis, key string) []string {
	t.Helper()
	members, err := mr.Members(key)
	if err != nil {
		t.Fatalf("members: %v", err)
	}
	return members
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

// ADR-0003: other services read sessions straight from Redis through the
// sessionclient module. Read what UM writes with that client, so a change to
// either side fails here.
func TestSessionStorageContract(t *testing.T) {
	res, mr := newRedis(t)
	store := NewSessionEntity(res)
	id, _ := store.CreateSession("u1", time.Hour, SessionMetadata{System: "PHARMACY"})

	rdb, err := sessionclient.NewRedisClient(mr.Addr())
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	defer rdb.Close()
	reader := sessionclient.NewRedisStore(rdb)
	ctx := context.Background()

	got, err := reader.Session(ctx, id)
	if err != nil || got != (sessionclient.Session{UserId: "u1", System: "PHARMACY"}) {
		t.Fatalf("sessionclient read %+v err=%v", got, err)
	}
	if ttl := mr.TTL(sessionclient.KeyPrefix + id); ttl != time.Hour {
		t.Fatalf("expected key to expire with the session, got %v", ttl)
	}

	checker := sessionclient.NewChecker(reader)
	if err := checker.Check(ctx, id, "PHARMACY"); err != nil {
		t.Fatalf("live session rejected: %v", err)
	}
	if err := store.RemoveSessionById(id); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := reader.Session(ctx, id); !errors.Is(err, sessionclient.ErrSessionRejected) {
		t.Fatalf("revoked session must read as rejected, got %v", err)
	}
}
