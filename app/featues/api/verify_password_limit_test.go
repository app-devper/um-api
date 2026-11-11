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

// countingUsers serves one User and counts how often the password check
// reached the user store.
type countingUsers struct {
	repository.IUser
	user    *model.User
	lookups int
}

func (u *countingUsers) GetUserById(string) (*model.User, error) {
	u.lookups++
	return u.user, nil
}

// verifyPasswordLimit is how many confirmations one address gets per minute.
const verifyPasswordLimit = 5

// The lockout is optional (LOGIN_LOCKOUT_ENABLED) and its check is not atomic,
// so the route must bound guessing by itself: with the lockout off, the sixth
// confirmation in a minute is refused before any password is compared.
func TestVerifyPasswordIsRateLimitedEvenWithTheLockoutOff(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	hashed, err := utils.HashPassword("right-password")
	if err != nil {
		t.Fatal(err)
	}
	user := &model.User{Id: primitive.NewObjectID(), Username: "alice", Password: hashed, ClientId: "123", Role: constant.ADMIN, Status: constant.ACTIVE}
	users := &countingUsers{user: user}
	lockoutOff := repository.NewLoginGuardEntity(&db.Resource{RdDB: rdb}, false)
	sessions := session.NewManager("test-secret", nil, users, session.WithLogin(nil, lockoutOff))
	r := gin.New()
	public := r.Group("/api/um/v1")
	signedIn := public.Group("", func(ctx *gin.Context) { ctx.Set(middlewares.UserId, user.Id.Hex()) })
	ApplyAuthAPI(public, signedIn, sessions, nil, nil, nil, nil, nil, rdb)
	confirm := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/um/v1/auth/verify-password", strings.NewReader(`{"password":"wrong","objective":"x"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}

	for i := 1; i <= verifyPasswordLimit; i++ {
		if w := confirm(); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), errs.ErrWrongPassword) {
			t.Fatalf("attempt %d: expected 400 %s, got %d %s", i, errs.ErrWrongPassword, w.Code, w.Body.String())
		}
	}
	w := confirm()

	if w.Code != http.StatusTooManyRequests || !strings.Contains(w.Body.String(), errs.ErrRateLimited) {
		t.Fatalf("attempt %d: expected 429 %s, got %d %s", verifyPasswordLimit+1, errs.ErrRateLimited, w.Code, w.Body.String())
	}
	if users.lookups != verifyPasswordLimit {
		t.Fatalf("the refused attempt still reached the password check: %d lookups, want %d", users.lookups, verifyPasswordLimit)
	}
}
