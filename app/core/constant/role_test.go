package constant

import (
	"testing"

	"github.com/app-devper/um-api/sessionclient"
)

// Services authorize with sessionclient's roles, so UM's must be the same
// values in the same order.
func TestRolesMatchSessionclient(t *testing.T) {
	pairs := []struct {
		um     string
		client sessionclient.Role
	}{{USER, sessionclient.RoleUser}, {MANAGER, sessionclient.RoleManager}, {ADMIN, sessionclient.RoleAdmin}, {SUPER, sessionclient.RoleSuper}}
	for i, p := range pairs {
		if p.um != string(p.client) {
			t.Errorf("UM role %q != sessionclient %q", p.um, p.client)
		}
		if i > 0 && !p.client.AtLeast(pairs[i-1].client) || i > 0 && pairs[i-1].client.AtLeast(p.client) {
			t.Errorf("%s must rank above %s", p.client, pairs[i-1].client)
		}
	}
}
