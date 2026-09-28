package useradmin

import (
	"um/app/core/constant"
	"um/app/domain/model"
)

// Permissions is what an Actor may do to one User: the same rules the
// commands in this package enforce, reported so clients render them instead
// of re-deriving role order (ADR-0006).
type Permissions struct {
	Edit        bool `json:"edit"`
	Delete      bool `json:"delete"`
	SetStatus   bool `json:"setStatus"`
	SetRole     bool `json:"setRole"`
	SetPassword bool `json:"setPassword"`
	Unlock      bool `json:"unlock"`
	// AssignableRoles are the Roles SetRole accepts for this User.
	AssignableRoles []string `json:"assignableRoles"`
}

// ManagedUser is a User as the Actor sees it: with what they may do to it.
type ManagedUser struct {
	model.User
	Can Permissions `json:"can"`
}

// Rules is what an Actor may do beyond individual Users.
type Rules struct {
	// CreatableRoles are the Roles Create accepts from this Actor.
	CreatableRoles []string `json:"creatableRoles"`
}

// Permit reports what actor may do to u.
func Permit(actor Actor, u model.User) Permissions {
	admin := actor.is(constant.SUPER, constant.ADMIN)
	self := u.Id.Hex() == actor.UserId
	manage := admin && actor.outranks(u.Role)
	p := Permissions{
		Edit:            admin && (self || manage),
		Delete:          manage && !self,
		SetStatus:       manage && !self,
		SetPassword:     manage,
		Unlock:          manage,
		AssignableRoles: []string{},
	}
	if manage {
		p.AssignableRoles = assignableRoles(actor, u)
		p.SetRole = len(p.AssignableRoles) > 0
	}
	return p
}

// assignableRoles are the Roles actor may give u, whom they manage.
func assignableRoles(actor Actor, u model.User) []string {
	switch actor.Role {
	case constant.SUPER:
		if u.ClientId == constant.SuperClientId {
			return []string{constant.SUPER, constant.ADMIN, constant.MANAGER, constant.USER}
		}
		return []string{constant.ADMIN, constant.MANAGER, constant.USER}
	case constant.ADMIN:
		return []string{constant.MANAGER, constant.USER}
	}
	return []string{}
}

// RulesFor reports what actor may do beyond individual Users.
func RulesFor(actor Actor) Rules {
	roles := []string{}
	for _, r := range []string{constant.SUPER, constant.ADMIN, constant.MANAGER, constant.USER} {
		if _, err := createTargetRole(actor.Role, r); err == nil {
			roles = append(roles, r)
		}
	}
	return Rules{CreatableRoles: roles}
}

// ListManaged lists the Users the Actor may see, each with its Permissions.
func (a *Admin) ListManaged(actor Actor) ([]ManagedUser, error) {
	users, err := a.List(actor)
	if err != nil {
		return nil, err
	}
	out := make([]ManagedUser, 0, len(users))
	for _, u := range users {
		out = append(out, ManagedUser{User: u, Can: Permit(actor, u)})
	}
	return out, nil
}

// GetManaged loads one User the Actor may see, with its Permissions.
func (a *Admin) GetManaged(actor Actor, id string) (*ManagedUser, error) {
	u, err := a.Get(actor, id)
	if err != nil {
		return nil, err
	}
	return &ManagedUser{User: *u, Can: Permit(actor, *u)}, nil
}
