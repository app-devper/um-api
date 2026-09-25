// Package useradmin decides which Actor may do what to which User, in which
// Client, and keeps the Session revocation that must follow status, role,
// password changes, and deletion. Other services (ADR-0003) treat a live
// Session as proof that its token's role and client are current, so every
// change to those must revoke. Denials are *errs.AppError values; anything else is a
// repository failure.
package useradmin

import (
	"um/app/core/constant"
	"um/app/core/errs"
	"um/app/core/utils"
	"um/app/domain/model"
	"um/app/domain/repository"
	"um/app/featues/request"

	"github.com/sirupsen/logrus"
)

// Actor is the authenticated user performing an operation, with their
// current Role and Client.
type Actor struct {
	UserId    string
	SessionId string
	Role      string
	ClientId  string
}

type Admin struct {
	users    repository.IUser
	sessions repository.ISession
	systems  repository.ISystem
	guard    repository.ILoginGuard
}

func New(users repository.IUser, sessions repository.ISession, systems repository.ISystem, guard repository.ILoginGuard) *Admin {
	return &Admin{users: users, sessions: sessions, systems: systems, guard: guard}
}

var (
	errNoPermission   = errs.New(errs.ErrNoPermission, "Don't have permission")
	errRolePermission = errs.New(errs.ErrInvalidRolePermission, "invalid role permission")
	errInvalidClient  = errs.New(errs.ErrInvalidClientId, "invalid client id")
	errInvalidRole    = errs.New(errs.ErrInvalidRole, "invalid role")
	errSuperClient    = errs.New(errs.ErrInvalidClientId, "SUPER must have clientId "+constant.SuperClientId)
)

func (a *Admin) List(actor Actor) ([]model.User, error) {
	switch actor.Role {
	case constant.SUPER:
		return a.users.GetUsers()
	case constant.ADMIN, constant.MANAGER:
		return a.users.GetUserAll(actor.ClientId)
	}
	return nil, errNoPermission
}

func (a *Admin) Get(actor Actor, id string) (*model.User, error) {
	if !actor.is(constant.SUPER, constant.ADMIN, constant.MANAGER) {
		return nil, errNoPermission
	}
	return a.users.GetUserByClientId(id, actor.scope())
}

func (a *Admin) Create(actor Actor, form request.User) (*model.User, error) {
	targetRole, err := createTargetRole(actor.Role, form.Role)
	if err != nil {
		return nil, err
	}

	switch actor.Role {
	case constant.SUPER:
		if len(form.ClientId) != 3 {
			return nil, errInvalidClient
		}
		if targetRole == constant.SUPER && form.ClientId != constant.SuperClientId {
			return nil, errSuperClient
		}
	case constant.ADMIN:
		if len(form.ClientId) != 3 || form.ClientId != actor.ClientId {
			return nil, errInvalidClient
		}
	}

	// Bootstrap exception: SUPER in clientId=000 doesn't require an existing system
	if !(targetRole == constant.SUPER && form.ClientId == constant.SuperClientId) {
		systems, err := a.systems.GetSystemsByClientId(form.ClientId)
		if err != nil {
			return nil, err
		}
		if len(systems) == 0 {
			return nil, errInvalidClient
		}
	}

	if found, _ := a.users.GetUserByUsername(form.Username); found != nil {
		return nil, errs.New(errs.ErrUsernameTaken, "username is taken")
	}

	form.CreatedBy = actor.UserId
	return a.users.CreateUser(form, targetRole)
}

func (a *Admin) Update(actor Actor, id string, form request.UpdateUser) (*model.User, error) {
	if !actor.is(constant.SUPER, constant.ADMIN) {
		return nil, errNoPermission
	}
	if id != actor.UserId {
		if _, err := a.manageable(actor, id); err != nil {
			return nil, err
		}
	}
	form.UpdatedBy = actor.UserId
	return a.users.UpdateUserById(id, actor.scope(), form)
}

// Delete removes a User and revokes all of their Sessions.
func (a *Admin) Delete(actor Actor, id string) (*model.User, error) {
	if !actor.is(constant.SUPER, constant.ADMIN) {
		return nil, errNoPermission
	}
	if id == actor.UserId {
		return nil, errs.New(errs.ErrDeleteSelf, "can't delete self user")
	}
	if _, err := a.manageable(actor, id); err != nil {
		return nil, err
	}
	result, err := a.users.RemoveUserById(id, actor.scope())
	if err != nil {
		return nil, err
	}
	a.revokeAll(id, "user deletion")
	return result, nil
}

// SetStatus changes a User's status and revokes all of their Sessions.
func (a *Admin) SetStatus(actor Actor, id string, form request.UpdateStatus) (*model.User, error) {
	if !actor.is(constant.SUPER, constant.ADMIN) {
		return nil, errNoPermission
	}
	if form.Status != constant.ACTIVE && form.Status != constant.INACTIVE {
		return nil, errs.New(errs.ErrBadRequest, "invalid status")
	}
	if id == actor.UserId {
		return nil, errs.New(errs.ErrDeleteSelf, "can't change status of self user")
	}
	if _, err := a.manageable(actor, id); err != nil {
		return nil, err
	}
	form.UpdatedBy = actor.UserId
	result, err := a.users.UpdateStatusById(id, actor.scope(), form)
	if err != nil {
		return nil, err
	}
	a.revokeAll(id, "status change")
	return result, nil
}

// SetRole changes a User's Role and revokes all of their Sessions.
func (a *Admin) SetRole(actor Actor, id string, form request.UpdateRole) (*model.User, error) {
	if !actor.is(constant.SUPER, constant.ADMIN) {
		return nil, errNoPermission
	}
	switch form.Role {
	case constant.SUPER, constant.ADMIN, constant.MANAGER, constant.USER:
	default:
		return nil, errInvalidRole
	}
	if form.Role == constant.SUPER && actor.Role != constant.SUPER {
		return nil, errInvalidRole
	}
	if actor.Role == constant.ADMIN && form.Role != constant.MANAGER && form.Role != constant.USER {
		return nil, errNoPermission
	}
	target, err := a.manageable(actor, id)
	if err != nil {
		return nil, err
	}
	if form.Role == constant.SUPER && target.ClientId != constant.SuperClientId {
		return nil, errSuperClient
	}
	form.UpdatedBy = actor.UserId
	result, err := a.users.UpdateRoleById(id, actor.scope(), form)
	if err != nil {
		return nil, err
	}
	a.revokeAll(id, "role change")
	return result, nil
}

// SetPassword resets another User's password and revokes all of their Sessions.
func (a *Admin) SetPassword(actor Actor, id string, form request.SetPassword) (*model.User, error) {
	if !actor.is(constant.SUPER, constant.ADMIN) {
		return nil, errNoPermission
	}
	if _, err := a.manageable(actor, id); err != nil {
		return nil, err
	}
	form.UpdatedBy = actor.UserId
	result, err := a.users.SetPassword(id, actor.scope(), form)
	if err != nil {
		return nil, err
	}
	a.revokeAll(id, "admin password reset")
	return result, nil
}

func (a *Admin) Unlock(actor Actor, id string) error {
	if !actor.is(constant.SUPER, constant.ADMIN) {
		return errNoPermission
	}
	target, err := a.manageable(actor, id)
	if err != nil {
		return err
	}
	return a.guard.Unlock(target.Username)
}

// ChangeOwnPassword changes the Actor's password and revokes their other
// Sessions, keeping the one making the request.
func (a *Admin) ChangeOwnPassword(actor Actor, form request.ChangePassword) (*model.User, error) {
	user, err := a.users.GetUserById(actor.UserId)
	if err != nil {
		return nil, err
	}
	if user == nil || utils.ComparePasswordAndHashedPassword(form.OldPassword, user.Password) != nil {
		return nil, errs.New(errs.ErrWrongPassword, "wrong password")
	}
	if user.Status != constant.ACTIVE {
		return nil, errs.New(errs.ErrTokenInvalid, "token invalid")
	}
	result, err := a.users.ChangePassword(user.Id.Hex(), actor.ClientId, form)
	if err != nil {
		return nil, err
	}
	if _, err := a.sessions.RevokeOtherSessions(user.Id.Hex(), actor.SessionId); err != nil {
		logrus.Warn("revoke other sessions after password change: ", err)
	}
	return result, nil
}

// manageable loads a target User within the Actor's scope and checks the
// Actor's Role outranks it.
func (a *Admin) manageable(actor Actor, id string) (*model.User, error) {
	target, err := a.users.GetUserByClientId(id, actor.scope())
	if err != nil {
		return nil, err
	}
	switch actor.Role {
	case constant.SUPER:
		if target.Role == constant.SUPER {
			return nil, errRolePermission
		}
	case constant.ADMIN:
		if target.Role == constant.SUPER || target.Role == constant.ADMIN {
			return nil, errRolePermission
		}
	default:
		return nil, errRolePermission
	}
	return target, nil
}

func (a *Admin) revokeAll(userId, reason string) {
	if _, err := a.sessions.RevokeOtherSessions(userId, ""); err != nil {
		logrus.Warnf("revoke sessions after %s: %v", reason, err)
	}
}

// scope is the Client filter for user lookups; empty means every Client.
func (actor Actor) scope() string {
	if actor.Role == constant.SUPER {
		return ""
	}
	return actor.ClientId
}

func (actor Actor) is(roles ...string) bool {
	for _, r := range roles {
		if actor.Role == r {
			return true
		}
	}
	return false
}

func createTargetRole(actorRole, requested string) (string, error) {
	switch actorRole {
	case constant.SUPER:
		switch requested {
		case "":
			return constant.ADMIN, nil
		case constant.SUPER, constant.ADMIN, constant.MANAGER, constant.USER:
			return requested, nil
		}
	case constant.ADMIN:
		switch requested {
		case "":
			return constant.USER, nil
		case constant.MANAGER, constant.USER:
			return requested, nil
		}
	}
	return "", errNoPermission
}
