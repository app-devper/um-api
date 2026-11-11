package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"um/app/core/constant"
	"um/app/core/errs"
	"um/app/core/utils"
	"um/app/domain/model"
	"um/app/domain/repository"
	"um/app/domain/session"
	"um/db"
	"um/middlewares"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// countingUsers serves Users by id and counts how often the password check
// reached the user store for each.
type countingUsers struct {
	repository.IUser
	byId    map[string]*model.User
	lookups map[string]int
}

func (u *countingUsers) GetUserById(id string) (*model.User, error) {
	u.lookups[id]++
	return u.byId[id], nil
}

// verifyPasswordRoute serves POST /auth/verify-password with the lockout off,
// for the two Users it returns; the request names its signed-in User in the
// X-Test-User header, standing in for RequireSession.
func verifyPasswordRoute(t *testing.T) (confirm func(userId string) *httptest.ResponseRecorder, users *countingUsers, alice, bob string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	hashed, err := utils.HashPassword("right-password")
	if err != nil {
		t.Fatal(err)
	}
	users = &countingUsers{byId: map[string]*model.User{}, lookups: map[string]int{}}
	for _, name := range []string{"alice", "bob"} {
		u := &model.User{Id: primitive.NewObjectID(), Username: name, Password: hashed, ClientId: "123", Role: constant.ADMIN, Status: constant.ACTIVE}
		users.byId[u.Id.Hex()] = u
		if name == "alice" {
			alice = u.Id.Hex()
		} else {
			bob = u.Id.Hex()
		}
	}
	lockoutOff := repository.NewLoginGuardEntity(&db.Resource{RdDB: rdb}, false)
	sessions := session.NewManager("test-secret", nil, users, session.WithLogin(nil, lockoutOff))
	r := gin.New()
	public := r.Group("/api/um/v1")
	signedIn := public.Group("", func(ctx *gin.Context) { ctx.Set(middlewares.UserId, ctx.GetHeader("X-Test-User")) })
	ApplyAuthAPI(public, signedIn, sessions, nil, nil, nil, nil, nil, rdb)
	confirm = func(userId string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/um/v1/auth/verify-password", strings.NewReader(`{"password":"wrong","objective":"x"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User", userId)
		r.ServeHTTP(w, req)
		return w
	}
	return confirm, users, alice, bob
}

// verifyPasswordLimit is how many confirmations one address gets per minute.
const verifyPasswordLimit = 5

// The lockout is optional (LOGIN_LOCKOUT_ENABLED) and its check is not atomic,
// so the route must bound guessing by itself: with the lockout off, the sixth
// confirmation in a minute is refused before any password is compared.
func TestVerifyPasswordIsRateLimitedEvenWithTheLockoutOff(t *testing.T) {
	confirm, users, alice, _ := verifyPasswordRoute(t)

	for i := 1; i <= verifyPasswordLimit; i++ {
		if w := confirm(alice); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), errs.ErrWrongPassword) {
			t.Fatalf("attempt %d: expected 400 %s, got %d %s", i, errs.ErrWrongPassword, w.Code, w.Body.String())
		}
	}
	w := confirm(alice)

	if w.Code != http.StatusTooManyRequests || !strings.Contains(w.Body.String(), errs.ErrRateLimited) {
		t.Fatalf("attempt %d: expected 429 %s, got %d %s", verifyPasswordLimit+1, errs.ErrRateLimited, w.Code, w.Body.String())
	}
	if users.lookups[alice] != verifyPasswordLimit {
		t.Fatalf("the refused attempt still reached the password check: %d lookups, want %d", users.lookups[alice], verifyPasswordLimit)
	}
}

// Behind the gateway every request reaches the service from one address, so a
// limit per address would let one User use up everyone's confirmations.
func TestVerifyPasswordLimitOfOneUserDoesNotRefuseAnother(t *testing.T) {
	confirm, _, alice, bob := verifyPasswordRoute(t)
	for i := 0; i <= verifyPasswordLimit; i++ {
		confirm(alice)
	}

	w := confirm(bob)

	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), errs.ErrWrongPassword) {
		t.Fatalf("bob shares alice's address but not her limit: expected 400 %s, got %d %s", errs.ErrWrongPassword, w.Code, w.Body.String())
	}
}
