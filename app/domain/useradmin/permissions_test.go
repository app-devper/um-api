package useradmin

import (
	"slices"
	"testing"

	"um/app/core/constant"
	"um/app/featues/request"
)

// What Permit reports is exactly what the commands allow, for every Actor
// Role against every target Role, and against the Actor itself.
func TestPermitMatchesTheCommands(t *testing.T) {
	type op struct {
		name   string
		report func(Permissions) bool
		run    func(a *Admin, actor Actor, id string) error
	}
	ops := []op{
		{"Edit", func(p Permissions) bool { return p.Edit }, func(a *Admin, actor Actor, id string) error {
			_, err := a.Update(actor, id, request.UpdateUser{FirstName: "Bob"})
			return err
		}},
		{"Delete", func(p Permissions) bool { return p.Delete }, func(a *Admin, actor Actor, id string) error {
			_, err := a.Delete(actor, id)
			return err
		}},
		{"SetStatus", func(p Permissions) bool { return p.SetStatus }, func(a *Admin, actor Actor, id string) error {
			_, err := a.SetStatus(actor, id, request.UpdateStatus{Status: constant.INACTIVE})
			return err
		}},
		{"SetPassword", func(p Permissions) bool { return p.SetPassword }, func(a *Admin, actor Actor, id string) error {
			_, err := a.SetPassword(actor, id, request.SetPassword{Password: "new-password-1"})
			return err
		}},
		{"Unlock", func(p Permissions) bool { return p.Unlock }, func(a *Admin, actor Actor, id string) error {
			return a.Unlock(actor, id)
		}},
	}
	for _, actorRole := range roles {
		for _, targetRole := range append(slices.Clone(roles), "self") {
			for _, o := range ops {
				f := newFixture()
				actorUser := f.user(actorRole, clientOf(actorRole))
				target := actorUser
				if targetRole != "self" {
					// Same Client as the Actor, so scoping never hides the target.
					target = f.user(targetRole, clientOf(actorRole))
				}
				actor := actorOf(actorUser)
				reported := o.report(Permit(actor, *target))
				allowed := o.run(f.admin, actor, target.Id.Hex()) == nil
				if reported != allowed {
					t.Errorf("%s %s → %s: Permit says %v, command allowed=%v", o.name, actorRole, targetRole, reported, allowed)
				}
			}
			// SetRole: the assignable roles are exactly the ones SetRole accepts.
			for _, want := range roles {
				f := newFixture()
				actorUser := f.user(actorRole, clientOf(actorRole))
				target := actorUser
				if targetRole != "self" {
					target = f.user(targetRole, clientOf(actorRole))
				}
				actor := actorOf(actorUser)
				p := Permit(actor, *target)
				reported := p.SetRole && slices.Contains(p.AssignableRoles, want)
				_, err := f.admin.SetRole(actor, target.Id.Hex(), request.UpdateRole{Role: want})
				if reported != (err == nil) {
					t.Errorf("SetRole %s → %s as %s: Permit says %v, command err=%v", actorRole, targetRole, want, reported, err)
				}
			}
		}
	}
}

// The creatable roles are exactly the ones Create accepts.
func TestRulesMatchCreate(t *testing.T) {
	for _, actorRole := range roles {
		f := newFixture()
		actor := actorOf(f.user(actorRole, clientOf(actorRole)))
		creatable := RulesFor(actor).CreatableRoles
		for _, want := range roles {
			client := "123"
			if actorRole == constant.SUPER && want == constant.SUPER {
				client = constant.SuperClientId
			}
			if actorRole == constant.ADMIN {
				client = actor.ClientId
			}
			_, err := f.admin.Create(actor, request.User{Username: "new" + actorRole + want, ClientId: client, Role: want})
			if slices.Contains(creatable, want) != (err == nil) {
				t.Errorf("Create %s as %s: rules say %v, command err=%v", want, actorRole, slices.Contains(creatable, want), err)
			}
		}
	}
}
