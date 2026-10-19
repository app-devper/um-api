package usecase

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"um/app/core/constant"
	"um/app/core/errs"
	"um/app/core/utils"
	"um/app/domain/model"
	"um/app/domain/passkey"
	"um/app/domain/repository"
	"um/db"
	"um/middlewares"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type noPasskeys struct{}

func (noPasskeys) Create(model.Passkey) error                 { return nil }
func (noPasskeys) Get(string) (*model.Passkey, error)         { return nil, repository.ErrPasskeyNotFound }
func (noPasskeys) ListByUser(string) ([]model.Passkey, error) { return []model.Passkey{}, nil }
func (noPasskeys) RecordUse(string, []byte, time.Time) error  { return nil }
func (noPasskeys) Delete(string, string) error                { return repository.ErrPasskeyNotFound }
func (noPasskeys) DeleteAllForUser(string) error              { return nil }

func passkeyRouter(t *testing.T, service *passkey.Service, userId string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/login/options", PasskeyLoginOptions(service))
	r.POST("/login", PasskeyLogin(service))
	authed := r.Group("", func(ctx *gin.Context) { ctx.Set(middlewares.UserId, userId) })
	authed.POST("/passkeys/options", PasskeyRegistrationOptions(service))
	authed.DELETE("/passkeys/:id", DeletePasskey(service))
	return r
}

func send(r *gin.Engine, method, path string, body any) (int, string) {
	encoded, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewReader(encoded))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	var parsed struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &parsed)
	return w.Code, parsed.Code
}

func enabledPasskeys(t *testing.T, user *model.User) *passkey.Service {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	service, err := passkey.New(passkey.Config{RPID: "devper.app", Origins: []string{"https://port.devper.app"}},
		noPasskeys{}, repository.NewPasskeyCeremonyEntity(&db.Resource{RdDB: client}), &authTestUserRepo{user: user}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestPasskeyRoutesAnswer404WhenDisabled(t *testing.T) {
	disabled, _ := passkey.New(passkey.Config{}, nil, nil, nil, nil)
	r := passkeyRouter(t, disabled, "u1")
	if status, code := send(r, http.MethodPost, "/login/options", gin.H{"system": "PORT"}); status != http.StatusNotFound || code != errs.ErrPasskeysDisabled {
		t.Fatalf("got %d %s", status, code)
	}
}

func TestPasskeyRouteErrors(t *testing.T) {
	hash, _ := utils.HashPassword("right-password")
	user := &model.User{Id: primitive.NewObjectID(), Username: "alice", Password: hash, ClientId: "123", Role: constant.USER, Status: constant.ACTIVE}
	r := passkeyRouter(t, enabledPasskeys(t, user), user.Id.Hex())
	cases := []struct {
		name, method, path string
		body               any
		status             int
		code               string
	}{
		{"login options without system", http.MethodPost, "/login/options", gin.H{}, http.StatusBadRequest, errs.ErrBadRequest},
		{"unknown ceremony", http.MethodPost, "/login", gin.H{"ceremonyId": "nope", "credential": gin.H{}}, http.StatusBadRequest, errs.ErrCeremonyExpired},
		{"registration with wrong password", http.MethodPost, "/passkeys/options", gin.H{"password": "wrong"}, http.StatusBadRequest, errs.ErrWrongPassword},
		{"delete a passkey that is not yours", http.MethodDelete, "/passkeys/abc", nil, http.StatusNotFound, errs.ErrNotFound},
	}
	for _, c := range cases {
		if status, code := send(r, c.method, c.path, c.body); status != c.status || code != c.code {
			t.Errorf("%s: got %d %s, want %d %s", c.name, status, code, c.status, c.code)
		}
	}
	status, _ := send(r, http.MethodPost, "/login/options", gin.H{"system": "PORT"})
	if status != http.StatusOK {
		t.Fatalf("login options: %d", status)
	}
	status, _ = send(r, http.MethodPost, "/passkeys/options", gin.H{"password": "right-password"})
	if status != http.StatusOK {
		t.Fatalf("registration options: %d", status)
	}
}
