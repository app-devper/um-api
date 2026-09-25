package session

import (
	"errors"
	"strings"
	"testing"
	"time"
	"um/app/core/config"
	"um/app/core/constant"
	"um/app/domain/model"
	"um/app/domain/repository"
	"um/db"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// redisStore is the real session store running on miniredis.
type redisStore struct {
	repository.ISession
	mr *miniredis.Miniredis
}

func (s *redisStore) ttl(sessionId string) time.Duration { return s.mr.TTL("session:" + sessionId) }
func (s *redisStore) exists(sessionId string) bool {
	_, err := s.GetSessionById(sessionId)
	return err == nil
}
func (s *redisStore) count() int {
	n := 0
	for _, key := range s.mr.Keys() {
		if strings.HasPrefix(key, "session:") {
			n++
		}
	}
	return n
}

type memUsers struct {
	repository.IUser
	byId map[string]*model.User
}

func (u *memUsers) GetUserById(id string) (*model.User, error) {
	if user, ok := u.byId[id]; ok {
		return user, nil
	}
	return nil, mongo.ErrNoDocuments
}

func setup(t *testing.T, secret string) (*Manager, *redisStore, *model.User) {
	t.Helper()
	user := &model.User{
		Id:       primitive.NewObjectID(),
		Username: "alice",
		ClientId: "123",
		Role:     constant.ADMIN,
		Status:   constant.ACTIVE,
	}
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := &redisStore{ISession: repository.NewSessionEntity(&db.Resource{RdDB: client}), mr: mr}
	users := &memUsers{byId: map[string]*model.User{user.Id.Hex(): user}}
	return NewManager(secret, store, users), store, user
}

func TestIssuedTokenVerifiesToPrincipal(t *testing.T) {
	m, store, user := setup(t, "test-secret")

	token, err := m.Issue(user, "UM", Metadata{})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	p, err := m.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if p.UserId != user.Id.Hex() || p.Role != constant.ADMIN || p.ClientId != "123" || p.System != "UM" {
		t.Fatalf("unexpected principal %+v", p)
	}
	if store.ttl(p.SessionId) != config.AccessTokenTime {
		t.Fatalf("expected session ttl %v, got %v", config.AccessTokenTime, store.ttl(p.SessionId))
	}
}

func TestVerifyUsesCurrentRoleAndClientNotTokenClaims(t *testing.T) {
	m, _, user := setup(t, "test-secret")
	token, _ := m.Issue(user, "UM", Metadata{})

	user.Role = constant.USER
	user.ClientId = "456"

	p, err := m.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if p.Role != constant.USER || p.ClientId != "456" {
		t.Fatalf("expected live role/client, got %+v", p)
	}
}

func TestVerifyRejectsRevokedSession(t *testing.T) {
	m, store, user := setup(t, "test-secret")
	token, _ := m.Issue(user, "UM", Metadata{})
	store.mr.FlushAll()

	if _, err := m.Verify(token); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("expected ErrSessionInvalid, got %v", err)
	}
}

func TestVerifyRejectsInactiveUser(t *testing.T) {
	m, _, user := setup(t, "test-secret")
	token, _ := m.Issue(user, "UM", Metadata{})
	user.Status = constant.INACTIVE

	if _, err := m.Verify(token); !errors.Is(err, ErrUserInactive) {
		t.Fatalf("expected ErrUserInactive, got %v", err)
	}
}

func TestVerifyRejectsDeletedUser(t *testing.T) {
	m, _, user := setup(t, "test-secret")
	token, _ := m.Issue(user, "UM", Metadata{})
	delete(m.users.(*memUsers).byId, user.Id.Hex())

	if _, err := m.Verify(token); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("expected ErrSessionInvalid, got %v", err)
	}
}

func TestVerifyRejectsBadTokens(t *testing.T) {
	m, store, user := setup(t, "test-secret")
	other := NewManager("other-secret", store, m.users)
	foreign, _ := other.Issue(user, "UM", Metadata{})

	unsigned, _ := jwt.NewWithClaims(jwt.SigningMethodNone, &accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{ID: "x"},
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)

	noId, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, &accessClaims{}).SignedString([]byte("test-secret"))

	for name, token := range map[string]string{
		"garbage":      "not-a-jwt",
		"wrong key":    foreign,
		"alg none":     unsigned,
		"no session":   noId,
		"empty string": "",
	} {
		if _, err := m.Verify(token); !errors.Is(err, ErrTokenInvalid) {
			t.Errorf("%s: expected ErrTokenInvalid, got %v", name, err)
		}
	}
}

func TestVerifyWithoutSecretFails(t *testing.T) {
	m, _, _ := setup(t, "")
	if _, err := m.Verify("anything"); !errors.Is(err, ErrNoSecret) {
		t.Fatalf("expected ErrNoSecret, got %v", err)
	}
}

func TestIssueRemovesSessionWhenSigningFails(t *testing.T) {
	m, store, user := setup(t, "")

	if _, err := m.Issue(user, "UM", Metadata{}); !errors.Is(err, ErrTokenGenFailed) {
		t.Fatalf("expected ErrTokenGenFailed, got %v", err)
	}
	if n := store.count(); n != 0 {
		t.Fatalf("expected no leaked session, got %d", n)
	}
}

func TestIssuePropagatesStoreFailure(t *testing.T) {
	m, store, user := setup(t, "test-secret")
	store.mr.SetError("redis down")

	_, err := m.Issue(user, "UM", Metadata{})
	if err == nil || errors.Is(err, ErrTokenGenFailed) {
		t.Fatalf("expected store error, got %v", err)
	}
}

func TestRenewExtendsSessionAndReturnsVerifiableToken(t *testing.T) {
	m, store, user := setup(t, "test-secret")
	token, _ := m.Issue(user, "UM", Metadata{})
	p, _ := m.Verify(token)
	store.mr.SetTTL("session:"+p.SessionId, time.Minute)

	renewed, err := m.Renew(p)
	if err != nil {
		t.Fatalf("renew: %v", err)
	}
	if store.ttl(p.SessionId) != config.AccessTokenTime {
		t.Fatalf("expected ttl reset to %v, got %v", config.AccessTokenTime, store.ttl(p.SessionId))
	}
	again, err := m.Verify(renewed)
	if err != nil || again.SessionId != p.SessionId {
		t.Fatalf("expected renewed token for same session, got %+v err=%v", again, err)
	}
}

func TestRenewRemovesSessionWhenSigningFails(t *testing.T) {
	signer, store, user := setup(t, "test-secret")
	token, _ := signer.Issue(user, "UM", Metadata{})
	p, _ := signer.Verify(token)
	broken := NewManager("", store, signer.users)

	if _, err := broken.Renew(p); !errors.Is(err, ErrTokenGenFailed) {
		t.Fatalf("expected ErrTokenGenFailed, got %v", err)
	}
	if store.exists(p.SessionId) {
		t.Fatal("expected session to be removed")
	}
}

func TestVerifyRejectsExpiredSession(t *testing.T) {
	m, store, user := setup(t, "test-secret")
	token, _ := m.Issue(user, "UM", Metadata{})

	store.mr.FastForward(config.AccessTokenTime + time.Second)

	if _, err := m.Verify(token); err == nil {
		t.Fatal("expected expired Session to fail verification")
	}
}

func TestVerifyRejectsSessionsRevokedByAdmin(t *testing.T) {
	m, store, user := setup(t, "test-secret")
	current, _ := m.Issue(user, "UM", Metadata{})
	other, _ := m.Issue(user, "POS", Metadata{})
	p, _ := m.Verify(current)

	if _, err := store.RevokeOtherSessions(user.Id.Hex(), p.SessionId); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	if _, err := m.Verify(current); err != nil {
		t.Fatalf("kept Session should verify, got %v", err)
	}
	if _, err := m.Verify(other); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("revoked Session should fail, got %v", err)
	}
}

func TestVerifyRejectsTokenForAnotherSystem(t *testing.T) {
	m, _, user := setup(t, "test-secret")
	token, _ := m.Issue(user, "UM", Metadata{})
	p, _ := m.Verify(token)

	p.System = "PHARMACY"
	forged, err := m.signJwt(*p)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if _, err := m.Verify(forged); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("expected ErrSessionInvalid for mismatched system, got %v", err)
	}
}
