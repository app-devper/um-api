package session

import (
	"errors"
	"testing"
	"um/app/core/constant"
	"um/app/core/utils"
	"um/app/domain/model"
	"um/app/domain/repository"
	"um/db"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// loginUsers adds lookup by username to memUsers.
type loginUsers struct{ memUsers }

func (u *loginUsers) GetUserByUsername(username string) (*model.User, error) {
	for _, user := range u.byId {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, mongo.ErrNoDocuments
}

// systems lists which Systems each Client has.
type systems struct {
	repository.ISystem
	byClient map[string][]string
}

func (s *systems) GetSystem(clientId, code string) (*model.System, error) {
	for _, c := range s.byClient[clientId] {
		if c == code {
			return &model.System{ClientId: clientId, SystemCode: code}, nil
		}
	}
	return nil, mongo.ErrNoDocuments
}

// brokenGuard is a lockout store that cannot be reached.
type brokenGuard struct{ repository.ILoginGuard }

func (brokenGuard) IsLocked(string) (bool, error) { return false, errors.New("redis down") }

// failingStore fails to end Sessions, as a session store that is down.
type failingStore struct {
	repository.ISession
	calls int
}

func (s *failingStore) RevokeOtherSessions(string, string) (int, error) {
	s.calls++
	return 0, errors.New("redis down")
}

type login struct {
	m     *Manager
	store *redisStore
	users *loginUsers
}

func newLogin(t *testing.T) *login {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	res := &db.Resource{RdDB: client}
	store := &redisStore{ISession: repository.NewSessionEntity(res), mr: mr}
	users := &loginUsers{memUsers{byId: map[string]*model.User{}}}
	sys := &systems{byClient: map[string][]string{"123": {"POS"}, constant.SuperClientId: {"SM", "POS"}}}
	m := NewManager("test-secret", store, users, WithLogin(sys, repository.NewLoginGuardEntity(res, true)))
	return &login{m: m, store: store, users: users}
}

func (l *login) user(t *testing.T, username, role, client, status string) *model.User {
	t.Helper()
	hash, err := utils.HashPassword("right-password")
	if err != nil {
		t.Fatal(err)
	}
	u := &model.User{Id: primitive.NewObjectID(), Username: username, Password: hash, Role: role, ClientId: client, Status: status}
	l.users.byId[u.Id.Hex()] = u
	return u
}

func (l *login) start(username, password, system string) (string, error) {
	return l.m.Start(Credentials{Username: username, Password: password, System: system}, Metadata{})
}

func TestStartIssuesAVerifiableSessionOnTheUsersSystem(t *testing.T) {
	l := newLogin(t)
	u := l.user(t, "alice", constant.ADMIN, "123", constant.ACTIVE)
	token, err := l.start(" alice ", "right-password", "POS")
	if err != nil {
		t.Fatal(err)
	}
	p, err := l.m.Verify(token)
	if err != nil || p.UserId != u.Id.Hex() || p.System != "POS" {
		t.Fatalf("principal %+v err %v", p, err)
	}
}

func TestStartRefusals(t *testing.T) {
	l := newLogin(t)
	l.user(t, "alice", constant.ADMIN, "123", constant.ACTIVE)
	l.user(t, "bob", constant.USER, "123", constant.INACTIVE)
	cases := []struct {
		name, user, password, system string
		want                         error
	}{
		{"unknown user", "nobody", "right-password", "POS", ErrWrongCredentials},
		{"wrong password", "alice", "wrong", "POS", ErrWrongCredentials},
		{"inactive", "bob", "right-password", "POS", ErrNotActive},
		{"system the client lacks", "alice", "right-password", "SM", ErrWrongSystem},
	}
	for _, c := range cases {
		if _, err := l.start(c.user, c.password, c.system); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
	if l.store.count() != 0 {
		t.Fatal("a refused login left a Session")
	}
}

// SUPER belongs to client 000 and is used only in System SM, even when
// client 000 has another System.
func TestSuperOnlyStartsASessionInSM(t *testing.T) {
	l := newLogin(t)
	l.user(t, "root", constant.SUPER, constant.SuperClientId, constant.ACTIVE)
	if _, err := l.start("root", "right-password", "POS"); !errors.Is(err, ErrWrongSystem) {
		t.Fatalf("SUPER on POS: %v", err)
	}
	if _, err := l.start("root", "right-password", "SM"); err != nil {
		t.Fatalf("SUPER on SM: %v", err)
	}
}

// A SUPER Session on another System, from before the rule, stops working.
func TestVerifyRefusesASuperSessionOutsideSM(t *testing.T) {
	l := newLogin(t)
	u := l.user(t, "root", constant.SUPER, constant.SuperClientId, constant.ACTIVE)
	token, err := l.m.Issue(u, "POS", Metadata{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.m.Verify(token); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("got %v", err)
	}
	if _, err := l.m.Resume(u.Id.Hex(), "POS", Metadata{}); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("resume: got %v", err)
	}
}

// Failures count toward the lockout, including a wrong System; success
// resets the count; a locked account is refused even with the right password.
func TestLockoutCountsFailuresAndResetsOnSuccess(t *testing.T) {
	l := newLogin(t)
	l.user(t, "alice", constant.ADMIN, "123", constant.ACTIVE)
	for i := 0; i < repository.LoginMaxFails-1; i++ {
		_, _ = l.start("alice", "wrong", "POS")
	}
	if _, err := l.start("alice", "right-password", "POS"); err != nil {
		t.Fatalf("under the limit: %v", err)
	}
	for i := 0; i < repository.LoginMaxFails-1; i++ {
		_, _ = l.start("alice", "wrong", "POS")
	}
	_, _ = l.start("alice", "right-password", "SM") // a wrong System counts too
	if _, err := l.start("alice", "right-password", "POS"); !errors.Is(err, ErrLocked) {
		t.Fatalf("after %d failures: %v", repository.LoginMaxFails, err)
	}
}

// If the lockout store cannot be reached no Session starts.
func TestStartFailsClosedWhenTheLockoutStoreIsDown(t *testing.T) {
	l := newLogin(t)
	l.user(t, "alice", constant.ADMIN, "123", constant.ACTIVE)
	WithLogin(&systems{byClient: map[string][]string{"123": {"POS"}}}, brokenGuard{})(l.m)
	if _, err := l.start("alice", "right-password", "POS"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("got %v", err)
	}
	if l.store.count() != 0 {
		t.Fatal("a Session started without the lockout check")
	}
}

func TestEndAllReportsSessionsItCouldNotEnd(t *testing.T) {
	store := &failingStore{}
	m := NewManager("test-secret", store, nil)
	if _, err := m.EndAll("u1", ""); !errors.Is(err, ErrNotEnded) {
		t.Fatalf("got %v", err)
	}
	if store.calls != endAttempts {
		t.Fatalf("tried %d times, want %d", store.calls, endAttempts)
	}
}

func TestEndAllKeepsTheCurrentSession(t *testing.T) {
	l := newLogin(t)
	l.user(t, "alice", constant.ADMIN, "123", constant.ACTIVE)
	keep, _ := l.start("alice", "right-password", "POS")
	other, _ := l.start("alice", "right-password", "POS")
	kp, _ := l.m.Verify(keep)
	if n, err := l.m.EndAll(kp.UserId, kp.SessionId); err != nil || n != 1 {
		t.Fatalf("ended %d, err %v", n, err)
	}
	if _, err := l.m.Verify(other); err == nil {
		t.Fatal("the other Session is still live")
	}
	if _, err := l.m.Verify(keep); err != nil {
		t.Fatalf("the kept Session ended: %v", err)
	}
}

func TestEndOtherChecksOwnership(t *testing.T) {
	l := newLogin(t)
	l.user(t, "alice", constant.ADMIN, "123", constant.ACTIVE)
	l.user(t, "carol", constant.USER, "123", constant.ACTIVE)
	a, _ := l.start("alice", "right-password", "POS")
	c, _ := l.start("carol", "right-password", "POS")
	ap, _ := l.m.Verify(a)
	cp, _ := l.m.Verify(c)
	if err := l.m.EndOther(ap.UserId, ap.SessionId, cp.SessionId); !errors.Is(err, ErrNotOwner) {
		t.Fatalf("other user's session: %v", err)
	}
	if err := l.m.EndOther(ap.UserId, ap.SessionId, ap.SessionId); !errors.Is(err, ErrCurrent) {
		t.Fatalf("current session: %v", err)
	}
	if err := l.m.EndOther(ap.UserId, ap.SessionId, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing session: %v", err)
	}
}
