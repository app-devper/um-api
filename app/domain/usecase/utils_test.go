package usecase

import (
	"net/http/httptest"
	"testing"
	"um/app/core/constant"
	"um/app/domain/model"
	"um/middlewares"

	"github.com/gin-gonic/gin"
)

func TestClientIdForRoleSuperReturnsEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.Role, constant.SUPER)
	c.Set(middlewares.ClientId, "000")
	if got := clientIdForRole(c); got != "" {
		t.Fatalf("SUPER should not scope by clientId, got %q", got)
	}
}

func TestClientIdForRoleNonSuperReturnsContext(t *testing.T) {
	cases := []string{constant.ADMIN, constant.MANAGER, constant.USER}
	for _, role := range cases {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set(middlewares.Role, role)
		c.Set(middlewares.ClientId, "123")
		if got := clientIdForRole(c); got != "123" {
			t.Fatalf("%s should scope by ctx clientId, got %q", role, got)
		}
	}
}

func TestValidateUserRoleSuperCannotManageSuper(t *testing.T) {
	if err := validateUserRole(constant.SUPER, &model.User{Role: constant.SUPER}); err == nil {
		t.Fatal("SUPER should not be allowed to manage another SUPER")
	}
	if err := validateUserRole(constant.SUPER, &model.User{Role: constant.ADMIN}); err != nil {
		t.Fatalf("SUPER should be allowed to manage ADMIN, got %v", err)
	}
	if err := validateUserRole(constant.SUPER, &model.User{Role: constant.USER}); err != nil {
		t.Fatalf("SUPER should be allowed to manage USER, got %v", err)
	}
}

func TestValidateUserRoleAdminBoundaries(t *testing.T) {
	if err := validateUserRole(constant.ADMIN, &model.User{Role: constant.SUPER}); err == nil {
		t.Fatal("ADMIN should not be allowed to manage SUPER")
	}
	if err := validateUserRole(constant.ADMIN, &model.User{Role: constant.ADMIN}); err == nil {
		t.Fatal("ADMIN should not be allowed to manage another ADMIN")
	}
	if err := validateUserRole(constant.ADMIN, &model.User{Role: constant.MANAGER}); err != nil {
		t.Fatalf("ADMIN should be allowed to manage MANAGER, got %v", err)
	}
	if err := validateUserRole(constant.ADMIN, &model.User{Role: constant.USER}); err != nil {
		t.Fatalf("ADMIN should be allowed to manage USER, got %v", err)
	}
}

func TestValidateUserRoleManagerAndUserForbidden(t *testing.T) {
	for _, caller := range []string{constant.MANAGER, constant.USER, "UNKNOWN"} {
		for _, target := range []string{constant.SUPER, constant.ADMIN, constant.MANAGER, constant.USER} {
			if err := validateUserRole(caller, &model.User{Role: target}); err == nil {
				t.Fatalf("%s must not be allowed to manage %s", caller, target)
			}
		}
	}
}

func TestResolveCreateTargetRoleSuperDefaultsToAdmin(t *testing.T) {
	role, err := resolveCreateTargetRole(constant.SUPER, "")
	if err != nil || role != constant.ADMIN {
		t.Fatalf("SUPER default should be ADMIN, got role=%q err=%v", role, err)
	}
}

func TestResolveCreateTargetRoleSuperHonorsRequested(t *testing.T) {
	for _, want := range []string{constant.SUPER, constant.ADMIN, constant.MANAGER, constant.USER} {
		got, err := resolveCreateTargetRole(constant.SUPER, want)
		if err != nil || got != want {
			t.Fatalf("SUPER requesting %s should succeed, got role=%q err=%v", want, got, err)
		}
	}
}

func TestResolveCreateTargetRoleAdminDefaultsToUser(t *testing.T) {
	role, err := resolveCreateTargetRole(constant.ADMIN, "")
	if err != nil || role != constant.USER {
		t.Fatalf("ADMIN default should be USER, got role=%q err=%v", role, err)
	}
}

func TestResolveCreateTargetRoleAdminLimited(t *testing.T) {
	if _, err := resolveCreateTargetRole(constant.ADMIN, constant.SUPER); err == nil {
		t.Fatal("ADMIN should not create SUPER")
	}
	if _, err := resolveCreateTargetRole(constant.ADMIN, constant.ADMIN); err == nil {
		t.Fatal("ADMIN should not create ADMIN")
	}
	if role, err := resolveCreateTargetRole(constant.ADMIN, constant.MANAGER); err != nil || role != constant.MANAGER {
		t.Fatalf("ADMIN should create MANAGER, got %q err=%v", role, err)
	}
	if role, err := resolveCreateTargetRole(constant.ADMIN, constant.USER); err != nil || role != constant.USER {
		t.Fatalf("ADMIN should create USER, got %q err=%v", role, err)
	}
}

func TestResolveCreateTargetRoleManagerForbidden(t *testing.T) {
	for _, requested := range []string{"", constant.SUPER, constant.ADMIN, constant.MANAGER, constant.USER} {
		if _, err := resolveCreateTargetRole(constant.MANAGER, requested); err == nil {
			t.Fatalf("MANAGER should not be allowed to create role=%q", requested)
		}
		if _, err := resolveCreateTargetRole(constant.USER, requested); err == nil {
			t.Fatalf("USER should not be allowed to create role=%q", requested)
		}
	}
}
