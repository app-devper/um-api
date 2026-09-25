package useradmin

import (
	"errors"
	"testing"
	"time"
	"um/app/core/constant"
	"um/app/core/errs"
	"um/app/core/utils"
	"um/app/domain/model"
	"um/app/domain/repository"
	"um/app/featues/request"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// memUsers is an in-memory user store honouring the repository's Client
// scoping: an empty clientId matches every Client.
type memUsers struct {
	repository.IUser
	byId map[string]*model.User
}

func (m *memUsers) find(id, clientId string) (*model.User, error) {
	u, ok := m.byId[id]
	if !ok || (clientId != "" && u.ClientId != clientId) {
		return nil, mongo.ErrNoDocuments
	}
	return u, nil
}
func (m *memUsers) GetUsers() ([]model.User, error) { return m.list(""), nil }
func (m *memUsers) GetUserAll(clientId string) ([]model.User, error) {
	return m.list(clientId), nil
}
func (m *memUsers) list(clientId string) []model.User {
	items := []model.User{}
	for _, u := range m.byId {
		if clientId == "" || u.ClientId == clientId {
			items = append(items, *u)
		}
	}
	return items
}
func (m *memUsers) GetUserById(id string) (*model.User, error) { return m.find(id, "") }
func (m *memUsers) GetUserByClientId(id, clientId string) (*model.User, error) {
	return m.find(id, clientId)
}
func (m *memUsers) GetUserByUsername(username string) (*model.User, error) {
	for _, u := range m.byId {
		if u.Username == utils.NormalizeUsername(username) {
			return u, nil
		}
	}
	return nil, mongo.ErrNoDocuments
}
func (m *memUsers) CreateUser(form request.User, role string) (*model.User, error) {
	u := &model.User{Id: primitive.NewObjectID(), Username: form.Username, ClientId: form.ClientId, Role: role, Status: constant.ACTIVE}
	m.byId[u.Id.Hex()] = u
	return u, nil
}
func (m *memUsers) RemoveUserById(id, clientId string) (*model.User, error) {
	u, err := m.find(id, clientId)
	if err == nil {
		delete(m.byId, id)
	}
	return u, err
}
func (m *memUsers) UpdateUserById(id, clientId string, form request.UpdateUser) (*model.User, error) {
	u, err := m.find(id, clientId)
	if err == nil {
		u.FirstName = form.FirstName
	}
	return u, err
}
func (m *memUsers) UpdateStatusById(id, clientId string, form request.UpdateStatus) (*model.User, error) {
	u, err := m.find(id, clientId)
	if err == nil {
		u.Status = form.Status
	}
	return u, err
}
func (m *memUsers) UpdateRoleById(id, clientId string, form request.UpdateRole) (*model.User, error) {
	u, err := m.find(id, clientId)
	if err == nil {
		u.Role = form.Role
	}
	return u, err
}
func (m *memUsers) SetPassword(id, clientId string, form request.SetPassword) (*model.User, error) {
	u, err := m.find(id, clientId)
	if err == nil {
		u.Password, _ = utils.HashPassword(form.Password)
	}
	return u, err
}
func (m *memUsers) ChangePassword(id, clientId string, form request.ChangePassword) (*model.User, error) {
	u, err := m.find(id, clientId)
	if err == nil {
		u.Password, _ = utils.HashPassword(form.NewPassword)
	}
	return u, err
}

type revocation struct{ userId, kept string }

type memSessions struct {
	repository.ISession
	revoked []revocation
}

func (s *memSessions) RevokeOtherSessions(userId, current string) (int, error) {
	s.revoked = append(s.revoked, revocation{userId, current})
	return 1, nil
}

type memSystems struct {
	repository.ISystem
	clients map[string]bool
}

func (s *memSystems) GetSystemsByClientId(clientId string) ([]model.System, error) {
	if s.clients[clientId] {
		return []model.System{{ClientId: clientId, SystemCode: "UM"}}, nil
	}
	return nil, nil
}

type memGuard struct {
	repository.ILoginGuard
	unlocked []string
}

func (g *memGuard) Unlock(username string) error {
	g.unlocked = append(g.unlocked, username)
	return nil
}

type fixture struct {
	admin    *Admin
	users    *memUsers
	sessions *memSessions
	guard    *memGuard
}

func newFixture() *fixture {
	f := &fixture{
		users:    &memUsers{byId: map[string]*model.User{}},
		sessions: &memSessions{},
		guard:    &memGuard{},
	}
	systems := &memSystems{clients: map[string]bool{"123": true, "456": true}}
	f.admin = New(f.users, f.sessions, systems, f.guard)
	return f
}

func (f *fixture) user(role, clientId string) *model.User {
	u := &model.User{Id: primitive.NewObjectID(), Username: role + clientId + time.Now().String(), Role: role, ClientId: clientId, Status: constant.ACTIVE}
	f.users.byId[u.Id.Hex()] = u
	return u
}

func actorOf(u *model.User) Actor {
	return Actor{UserId: u.Id.Hex(), SessionId: "current", Role: u.Role, ClientId: u.ClientId}
}

func code(err error) string {
	var appErr *errs.AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	if err != nil {
		return err.Error()
	}
	return ""
}

var roles = []string{constant.SUPER, constant.ADMIN, constant.MANAGER, constant.USER}

// clientOf keeps SUPER in its own Client, matching the bootstrap rule.
func clientOf(role string) string {
	if role == constant.SUPER {
		return constant.SuperClientId
	}
	return "123"
}

func TestManagementGrid(t *testing.T) {
	want := map[string]map[string]string{
		constant.SUPER: {constant.SUPER: errs.ErrInvalidRolePermission, constant.ADMIN: "", constant.MANAGER: "", constant.USER: ""},
		// SUPER lives in Client 000, so an ADMIN can't even see it.
		constant.ADMIN:   {constant.SUPER: mongo.ErrNoDocuments.Error(), constant.ADMIN: errs.ErrInvalidRolePermission, constant.MANAGER: "", constant.USER: ""},
		constant.MANAGER: {constant.SUPER: errs.ErrNoPermission, constant.ADMIN: errs.ErrNoPermission, constant.MANAGER: errs.ErrNoPermission, constant.USER: errs.ErrNoPermission},
		constant.USER:    {constant.SUPER: errs.ErrNoPermission, constant.ADMIN: errs.ErrNoPermission, constant.MANAGER: errs.ErrNoPermission, constant.USER: errs.ErrNoPermission},
	}
	ops := []struct {
		name    string
		revokes bool
		run     func(a *Admin, actor Actor, id string) error
	}{
		{"SetStatus", true, func(a *Admin, actor Actor, id string) error {
			_, err := a.SetStatus(actor, id, request.UpdateStatus{Status: constant.INACTIVE})
			return err
		}},
		{"SetRole", true, func(a *Admin, actor Actor, id string) error {
			_, err := a.SetRole(actor, id, request.UpdateRole{Role: constant.USER})
			return err
		}},
		{"SetPassword", true, func(a *Admin, actor Actor, id string) error {
			_, err := a.SetPassword(actor, id, request.SetPassword{Password: "new-password-1"})
			return err
		}},
		{"Update", false, func(a *Admin, actor Actor, id string) error {
			_, err := a.Update(actor, id, request.UpdateUser{FirstName: "Bob"})
			return err
		}},
		{"Delete", true, func(a *Admin, actor Actor, id string) error {
			_, err := a.Delete(actor, id)
			return err
		}},
		{"Unlock", false, func(a *Admin, actor Actor, id string) error {
			return a.Unlock(actor, id)
		}},
	}
	for _, op := range ops {
		for _, actorRole := range roles {
			for _, targetRole := range roles {
				f := newFixture()
				actor := f.user(actorRole, clientOf(actorRole))
				target := f.user(targetRole, clientOf(targetRole))

				err := op.run(f.admin, actorOf(actor), target.Id.Hex())

				if got := code(err); got != want[actorRole][targetRole] {
					t.Errorf("%s %s → %s: expected %q, got %q", op.name, actorRole, targetRole, want[actorRole][targetRole], got)
				}
				revoked := len(f.sessions.revoked) == 1
				if revoked != (op.revokes && err == nil) {
					t.Errorf("%s %s → %s: revoked=%v with err=%v", op.name, actorRole, targetRole, revoked, err)
				}
			}
		}
	}
}

func TestAdminIsScopedToOwnClient(t *testing.T) {
	f := newFixture()
	admin := f.user(constant.ADMIN, "123")
	other := f.user(constant.USER, "456")

	if _, err := f.admin.Get(actorOf(admin), other.Id.Hex()); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("expected not found across Clients, got %v", err)
	}
	if _, err := f.admin.Delete(actorOf(admin), other.Id.Hex()); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("expected not found across Clients, got %v", err)
	}

	super := f.user(constant.SUPER, constant.SuperClientId)
	if _, err := f.admin.Get(actorOf(super), other.Id.Hex()); err != nil {
		t.Fatalf("SUPER should reach any Client, got %v", err)
	}
}

func TestListScoping(t *testing.T) {
	f := newFixture()
	super := f.user(constant.SUPER, constant.SuperClientId)
	admin := f.user(constant.ADMIN, "123")
	manager := f.user(constant.MANAGER, "123")
	user := f.user(constant.USER, "123")
	f.user(constant.USER, "456")

	all, _ := f.admin.List(actorOf(super))
	if len(all) != 5 {
		t.Fatalf("SUPER should list every Client, got %d", len(all))
	}
	for _, actor := range []*model.User{admin, manager} {
		scoped, _ := f.admin.List(actorOf(actor))
		if len(scoped) != 3 {
			t.Fatalf("%s should list own Client only, got %d", actor.Role, len(scoped))
		}
	}
	if _, err := f.admin.List(actorOf(user)); code(err) != errs.ErrNoPermission {
		t.Fatalf("USER should be denied, got %v", err)
	}
}

func TestCreateTargetRoles(t *testing.T) {
	cases := []struct {
		actor, requested, want, err string
	}{
		{constant.SUPER, "", constant.ADMIN, ""},
		{constant.SUPER, constant.ADMIN, constant.ADMIN, ""},
		{constant.SUPER, constant.MANAGER, constant.MANAGER, ""},
		{constant.SUPER, constant.USER, constant.USER, ""},
		{constant.ADMIN, "", constant.USER, ""},
		{constant.ADMIN, constant.MANAGER, constant.MANAGER, ""},
		{constant.ADMIN, constant.USER, constant.USER, ""},
		{constant.ADMIN, constant.ADMIN, "", errs.ErrNoPermission},
		{constant.ADMIN, constant.SUPER, "", errs.ErrNoPermission},
		{constant.MANAGER, "", "", errs.ErrNoPermission},
		{constant.MANAGER, constant.USER, "", errs.ErrNoPermission},
		{constant.USER, "", "", errs.ErrNoPermission},
	}
	for _, tc := range cases {
		f := newFixture()
		actor := f.user(tc.actor, clientOf(tc.actor))
		created, err := f.admin.Create(actorOf(actor), request.User{Username: "new-user", ClientId: "123", Role: tc.requested})
		if code(err) != tc.err {
			t.Errorf("%s creating %q: expected %q, got %v", tc.actor, tc.requested, tc.err, err)
			continue
		}
		if err == nil && created.Role != tc.want {
			t.Errorf("%s creating %q: expected role %s, got %s", tc.actor, tc.requested, tc.want, created.Role)
		}
	}
}

func TestCreateClientRules(t *testing.T) {
	f := newFixture()
	super := actorOf(f.user(constant.SUPER, constant.SuperClientId))
	admin := actorOf(f.user(constant.ADMIN, "123"))

	cases := []struct {
		name  string
		actor Actor
		form  request.User
		err   string
	}{
		{"admin in other Client", admin, request.User{Username: "a1", ClientId: "456"}, errs.ErrInvalidClientId},
		{"Client without systems", super, request.User{Username: "a2", ClientId: "789"}, errs.ErrInvalidClientId},
		{"SUPER outside 000", super, request.User{Username: "a3", ClientId: "123", Role: constant.SUPER}, errs.ErrInvalidClientId},
		{"SUPER bootstrap in 000", super, request.User{Username: "a4", ClientId: constant.SuperClientId, Role: constant.SUPER}, ""},
		{"bad Client length", super, request.User{Username: "a5", ClientId: "12"}, errs.ErrInvalidClientId},
		{"duplicate username", admin, request.User{Username: "a4", ClientId: "123"}, errs.ErrUsernameTaken},
	}
	for _, tc := range cases {
		if _, err := f.admin.Create(tc.actor, tc.form); code(err) != tc.err {
			t.Errorf("%s: expected %q, got %v", tc.name, tc.err, err)
		}
	}
}

func TestSetRoleRules(t *testing.T) {
	f := newFixture()
	super := actorOf(f.user(constant.SUPER, constant.SuperClientId))
	admin := actorOf(f.user(constant.ADMIN, "123"))
	target := f.user(constant.USER, "123").Id.Hex()

	cases := []struct {
		name  string
		actor Actor
		role  string
		err   string
	}{
		{"unknown role", super, "ROOT", errs.ErrInvalidRole},
		{"admin grants SUPER", admin, constant.SUPER, errs.ErrInvalidRole},
		{"admin grants ADMIN", admin, constant.ADMIN, errs.ErrNoPermission},
		{"SUPER outside 000", super, constant.SUPER, errs.ErrInvalidClientId},
		{"admin grants MANAGER", admin, constant.MANAGER, ""},
		{"super grants ADMIN", super, constant.ADMIN, ""},
	}
	for _, tc := range cases {
		before := len(f.sessions.revoked)
		_, err := f.admin.SetRole(tc.actor, target, request.UpdateRole{Role: tc.role})
		if code(err) != tc.err {
			t.Errorf("%s: expected %q, got %v", tc.name, tc.err, err)
		}
		if revoked := len(f.sessions.revoked) > before; revoked != (err == nil) {
			t.Errorf("%s: revoked=%v with err=%v", tc.name, revoked, err)
		}
	}
}

func TestSelfProtection(t *testing.T) {
	f := newFixture()
	admin := f.user(constant.ADMIN, "123")
	self := admin.Id.Hex()

	if _, err := f.admin.Delete(actorOf(admin), self); code(err) != errs.ErrDeleteSelf {
		t.Fatalf("expected delete-self denial, got %v", err)
	}
	if _, err := f.admin.SetStatus(actorOf(admin), self, request.UpdateStatus{Status: constant.INACTIVE}); code(err) != errs.ErrDeleteSelf {
		t.Fatalf("expected status-self denial, got %v", err)
	}
	if _, err := f.admin.Update(actorOf(admin), self, request.UpdateUser{FirstName: "Me"}); err != nil {
		t.Fatalf("ADMIN should update own profile despite outranking rule, got %v", err)
	}
}

func TestPasswordChangesRevokeSessions(t *testing.T) {
	f := newFixture()
	admin := f.user(constant.ADMIN, "123")
	target := f.user(constant.USER, "123")

	if _, err := f.admin.SetPassword(actorOf(admin), target.Id.Hex(), request.SetPassword{Password: "new-password-1"}); err != nil {
		t.Fatalf("set password: %v", err)
	}
	if got := f.sessions.revoked; len(got) != 1 || got[0] != (revocation{target.Id.Hex(), ""}) {
		t.Fatalf("expected every target Session revoked, got %+v", got)
	}

	target.Password, _ = utils.HashPassword("old-password")
	self := actorOf(target)
	if _, err := f.admin.ChangeOwnPassword(self, request.ChangePassword{OldPassword: "wrong", NewPassword: "new-password-2"}); code(err) != errs.ErrWrongPassword {
		t.Fatalf("expected wrong password, got %v", err)
	}
	if len(f.sessions.revoked) != 1 {
		t.Fatalf("wrong old password must not revoke, got %+v", f.sessions.revoked)
	}
	if _, err := f.admin.ChangeOwnPassword(self, request.ChangePassword{OldPassword: "old-password", NewPassword: "new-password-2"}); err != nil {
		t.Fatalf("change own password: %v", err)
	}
	if got := f.sessions.revoked[1]; got != (revocation{target.Id.Hex(), "current"}) {
		t.Fatalf("expected other Sessions revoked keeping current, got %+v", got)
	}
}

func TestUnlock(t *testing.T) {
	f := newFixture()
	admin := f.user(constant.ADMIN, "123")
	target := f.user(constant.USER, "123")
	peer := f.user(constant.ADMIN, "123")

	if err := f.admin.Unlock(actorOf(admin), target.Id.Hex()); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	if len(f.guard.unlocked) != 1 || f.guard.unlocked[0] != target.Username {
		t.Fatalf("expected guard unlock for %s, got %v", target.Username, f.guard.unlocked)
	}
	if err := f.admin.Unlock(actorOf(admin), peer.Id.Hex()); code(err) != errs.ErrInvalidRolePermission {
		t.Fatalf("ADMIN must not unlock a peer ADMIN, got %v", err)
	}
}
