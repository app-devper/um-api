package passkey

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
	"um/app/core/constant"
	"um/app/core/utils"
	"um/app/domain/model"
	"um/app/domain/repository"
	"um/app/domain/session"
	"um/db"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	rpID   = "devper.app"
	origin = "https://port.devper.app"
)

type users struct {
	repository.IUser
	byId map[string]*model.User
}

func (u *users) GetUserById(id string) (*model.User, error) {
	if user, ok := u.byId[id]; ok {
		return user, nil
	}
	return nil, mongo.ErrNoDocuments
}

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

type keys struct {
	byId map[string]model.Passkey
}

func (k *keys) Create(p model.Passkey) error {
	if _, ok := k.byId[p.Id]; ok {
		return errors.New("duplicate credential")
	}
	k.byId[p.Id] = p
	return nil
}

func (k *keys) Get(id string) (*model.Passkey, error) {
	p, ok := k.byId[id]
	if !ok {
		return nil, repository.ErrPasskeyNotFound
	}
	return &p, nil
}

func (k *keys) ListByUser(userId string) ([]model.Passkey, error) {
	out := []model.Passkey{}
	for _, p := range k.byId {
		if p.UserId == userId {
			out = append(out, p)
		}
	}
	return out, nil
}

func (k *keys) RecordUse(id string, credential []byte, at time.Time) error {
	p, ok := k.byId[id]
	if !ok {
		return repository.ErrPasskeyNotFound
	}
	p.Credential, p.LastUsedDate = credential, at
	k.byId[id] = p
	return nil
}

func (k *keys) Delete(id, userId string) error {
	p, ok := k.byId[id]
	if !ok || p.UserId != userId {
		return repository.ErrPasskeyNotFound
	}
	delete(k.byId, id)
	return nil
}

func (k *keys) DeleteAllForUser(userId string) error {
	for id, p := range k.byId {
		if p.UserId == userId {
			delete(k.byId, id)
		}
	}
	return nil
}

type fixture struct {
	t        *testing.T
	service  *Service
	sessions *session.Manager
	users    *users
	keys     *keys
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	res := &db.Resource{RdDB: client}
	u := &users{byId: map[string]*model.User{}}
	sys := &systems{byClient: map[string][]string{"123": {"PORT"}}}
	sessions := session.NewManager("test-secret", repository.NewSessionEntity(res), u, session.WithLogin(sys, repository.NewLoginGuardEntity(res, true)))
	k := &keys{byId: map[string]model.Passkey{}}
	service, err := New(Config{RPID: rpID, RPName: "Devper", Origins: []string{origin}}, k, repository.NewPasskeyCeremonyEntity(res), u, sessions)
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{t: t, service: service, sessions: sessions, users: u, keys: k}
}

func (f *fixture) user(username, status string) *model.User {
	f.t.Helper()
	hash, err := utils.HashPassword("right-password")
	if err != nil {
		f.t.Fatal(err)
	}
	user := &model.User{Id: primitive.NewObjectID(), Username: username, Password: hash, Role: constant.USER, ClientId: "123", Status: status}
	f.users.byId[user.Id.Hex()] = user
	return user
}

func (f *fixture) register(user *model.User, a *softAuthenticator) model.Passkey {
	f.t.Helper()
	options, err := f.service.BeginRegistration(user.Id.Hex(), "right-password")
	if err != nil {
		f.t.Fatal(err)
	}
	key, err := f.service.FinishRegistration(user.Id.Hex(), options.CeremonyId, "MacBook", a.create(options.PublicKey))
	if err != nil {
		f.t.Fatal(err)
	}
	return key
}

func (f *fixture) login(a *softAuthenticator, system string) (string, error) {
	f.t.Helper()
	options, err := f.service.BeginLogin(system)
	if err != nil {
		f.t.Fatal(err)
	}
	return f.service.FinishLogin(options.CeremonyId, a.get(options.PublicKey), session.Metadata{})
}

func TestRegisteredPasskeyStartsAVerifiableSession(t *testing.T) {
	f := newFixture(t)
	alice := f.user("alice", constant.ACTIVE)
	a := newSoftAuthenticator(t, rpID, origin)
	key := f.register(alice, a)
	if key.Name != "MacBook" || key.UserId != alice.Id.Hex() {
		t.Fatalf("stored %+v", key)
	}
	token, err := f.login(a, "PORT")
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.sessions.Verify(token)
	if err != nil || p.UserId != alice.Id.Hex() || p.System != "PORT" {
		t.Fatalf("principal %+v err %v", p, err)
	}
	if f.keys.byId[key.Id].LastUsedDate.IsZero() {
		t.Fatal("login did not record use")
	}
}

func TestRegistrationNeedsTheCurrentPassword(t *testing.T) {
	f := newFixture(t)
	alice := f.user("alice", constant.ACTIVE)
	if _, err := f.service.BeginRegistration(alice.Id.Hex(), "wrong"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("got %v", err)
	}
}

func TestACeremonyIsAnsweredOnceAndOnlyByItsUser(t *testing.T) {
	f := newFixture(t)
	alice := f.user("alice", constant.ACTIVE)
	bob := f.user("bob", constant.ACTIVE)
	a := newSoftAuthenticator(t, rpID, origin)
	options, err := f.service.BeginRegistration(alice.Id.Hex(), "right-password")
	if err != nil {
		t.Fatal(err)
	}
	response := a.create(options.PublicKey)
	if _, err := f.service.FinishRegistration(bob.Id.Hex(), options.CeremonyId, "x", response); !errors.Is(err, ErrCeremony) {
		t.Fatalf("other user: %v", err)
	}
	if _, err := f.service.FinishRegistration(alice.Id.Hex(), options.CeremonyId, "x", response); !errors.Is(err, ErrCeremony) {
		t.Fatalf("answered twice: %v", err)
	}

	f.register(alice, a)
	login, err := f.service.BeginLogin("PORT")
	if err != nil {
		t.Fatal(err)
	}
	assertion := a.get(login.PublicKey)
	if _, err := f.service.FinishLogin(login.CeremonyId, assertion, session.Metadata{}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.FinishLogin(login.CeremonyId, assertion, session.Metadata{}); !errors.Is(err, ErrCeremony) {
		t.Fatalf("login replayed: %v", err)
	}
}

func TestLoginRefusals(t *testing.T) {
	f := newFixture(t)
	alice := f.user("alice", constant.ACTIVE)
	a := newSoftAuthenticator(t, rpID, origin)
	f.register(alice, a)

	stranger := newSoftAuthenticator(t, rpID, origin)
	stranger.userHandle = []byte(alice.Id.Hex())
	if _, err := f.login(stranger, "PORT"); !errors.Is(err, ErrNotAccepted) {
		t.Errorf("unregistered credential: %v", err)
	}

	bob := f.user("bob", constant.ACTIVE)
	impostor := *a
	impostor.userHandle = []byte(bob.Id.Hex())
	if _, err := f.login(&impostor, "PORT"); !errors.Is(err, ErrNotAccepted) {
		t.Errorf("alice's credential claiming bob: %v", err)
	}

	presenceOnly := *a
	presenceOnly.skipUV = true
	if _, err := f.login(&presenceOnly, "PORT"); !errors.Is(err, ErrNotAccepted) {
		t.Errorf("assertion without user verification: %v", err)
	}

	phishing := *a
	phishing.origin = "https://devper-port.web.app"
	if _, err := f.login(&phishing, "PORT"); !errors.Is(err, ErrNotAccepted) {
		t.Errorf("origin outside the RP: %v", err)
	}

	if _, err := f.login(a, "SM"); !errors.Is(err, session.ErrWrongSystem) {
		t.Errorf("system the client lacks: %v", err)
	}

	alice.Status = constant.INACTIVE
	if _, err := f.login(a, "PORT"); !errors.Is(err, session.ErrNotActive) {
		t.Errorf("inactive user: %v", err)
	}
}

// A sign count that goes backwards means two copies of the key exist.
func TestASignCountThatGoesBackwardsIsRefused(t *testing.T) {
	f := newFixture(t)
	alice := f.user("alice", constant.ACTIVE)
	a := newSoftAuthenticator(t, rpID, origin)
	f.register(alice, a)
	a.signCount = 10
	if _, err := f.login(a, "PORT"); err != nil {
		t.Fatal(err)
	}
	a.signCount = 5
	if _, err := f.login(a, "PORT"); !errors.Is(err, ErrNotAccepted) {
		t.Fatalf("cloned authenticator: %v", err)
	}
}

func TestUsersListAndDeleteOnlyTheirOwnPasskeys(t *testing.T) {
	f := newFixture(t)
	alice := f.user("alice", constant.ACTIVE)
	bob := f.user("bob", constant.ACTIVE)
	key := f.register(alice, newSoftAuthenticator(t, rpID, origin))
	f.register(bob, newSoftAuthenticator(t, rpID, origin))

	list, err := f.service.List(alice.Id.Hex())
	if err != nil || len(list) != 1 || list[0].Id != key.Id {
		t.Fatalf("list %+v err %v", list, err)
	}
	if err := f.service.Delete(bob.Id.Hex(), key.Id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("bob deleted alice's passkey: %v", err)
	}
	if err := f.service.Delete(alice.Id.Hex(), key.Id); err != nil {
		t.Fatal(err)
	}
	if list, _ := f.service.List(alice.Id.Hex()); len(list) != 0 {
		t.Fatal("passkey still listed")
	}
}

func TestRegisteringTheSameAuthenticatorTwiceIsExcluded(t *testing.T) {
	f := newFixture(t)
	alice := f.user("alice", constant.ACTIVE)
	a := newSoftAuthenticator(t, rpID, origin)
	key := f.register(alice, a)
	options, err := f.service.BeginRegistration(alice.Id.Hex(), "right-password")
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		PublicKey struct {
			ExcludeCredentials []struct {
				ID string `json:"id"`
			} `json:"excludeCredentials"`
			AuthenticatorSelection struct {
				ResidentKey      string `json:"residentKey"`
				UserVerification string `json:"userVerification"`
			} `json:"authenticatorSelection"`
		} `json:"publicKey"`
	}
	if err := json.Unmarshal(options.PublicKey, &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.PublicKey.ExcludeCredentials) != 1 || parsed.PublicKey.ExcludeCredentials[0].ID != key.Id {
		t.Fatalf("exclusions %+v", parsed.PublicKey.ExcludeCredentials)
	}
	if parsed.PublicKey.AuthenticatorSelection.ResidentKey != "required" || parsed.PublicKey.AuthenticatorSelection.UserVerification != "required" {
		t.Fatalf("selection %+v", parsed.PublicKey.AuthenticatorSelection)
	}
}

func TestWithoutAnRPIDPasskeysAreDisabled(t *testing.T) {
	service, err := New(Config{}, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.BeginLogin("PORT"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("got %v", err)
	}
	if service.Enabled() {
		t.Fatal("reported enabled")
	}
}
