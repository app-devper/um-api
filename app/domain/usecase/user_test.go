package usecase

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"um/app/core/constant"
	"um/app/core/errs"
	"um/app/core/utils"
	"um/app/domain/model"
	"um/app/domain/repository"
	"um/app/domain/useradmin"
	"um/app/featues/request"
	"um/middlewares"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// userUseCaseRepo backs the adapter tests; policy lives in useradmin and is
// tested there.
type userUseCaseRepo struct {
	repository.IUser
	byId                 map[string]*model.User
	byUsername           map[string]*model.User
	changePasswordCall   string
	updateUserCall       string
	updateUserClient     string
	createdUser          *model.User
	getUsersCalled       bool
	getUserAllCalledWith string
}

func (r *userUseCaseRepo) GetUsers() ([]model.User, error) {
	r.getUsersCalled = true
	return nil, nil
}
func (r *userUseCaseRepo) GetUserAll(clientId string) ([]model.User, error) {
	r.getUserAllCalledWith = clientId
	return nil, nil
}
func (r *userUseCaseRepo) GetUserByUsername(username string) (*model.User, error) {
	if u, ok := r.byUsername[utils.NormalizeUsername(username)]; ok {
		return u, nil
	}
	return nil, mongo.ErrNoDocuments
}
func (r *userUseCaseRepo) GetUserById(id string) (*model.User, error) {
	if u, ok := r.byId[id]; ok {
		return u, nil
	}
	return nil, mongo.ErrNoDocuments
}
func (r *userUseCaseRepo) GetUserByClientId(id string, clientId string) (*model.User, error) {
	if u, ok := r.byId[id]; ok && (clientId == "" || u.ClientId == clientId) {
		return u, nil
	}
	return nil, mongo.ErrNoDocuments
}
func (r *userUseCaseRepo) CreateUser(form request.User, role string) (*model.User, error) {
	r.createdUser = &model.User{Id: primitive.NewObjectID(), Username: form.Username, ClientId: form.ClientId, Role: role}
	return r.createdUser, nil
}
func (r *userUseCaseRepo) UpdateUserById(id string, clientId string, form request.UpdateUser) (*model.User, error) {
	r.updateUserCall = id
	r.updateUserClient = clientId
	return r.byId[id], nil
}
func (r *userUseCaseRepo) ChangePassword(id string, clientId string, form request.ChangePassword) (*model.User, error) {
	r.changePasswordCall = id
	return r.byId[id], nil
}

func TestChangePasswordRevokesOtherSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hashed, err := utils.HashPassword("old-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	userID := primitive.NewObjectID()
	userRepo := &userUseCaseRepo{
		byId: map[string]*model.User{
			userID.Hex(): {
				Id:       userID,
				Username: "alice",
				Password: hashed,
				ClientId: "123",
				Role:     constant.USER,
				Status:   constant.ACTIVE,
			},
		},
	}
	sessionRepo := &authTestSessionRepo{revokeCount: 2}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{
		"oldPassword": "old-password",
		"newPassword": "new-password-1",
	})
	c.Request = httptest.NewRequest(http.MethodPut, "/user/change-password", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middlewares.UserId, userID.Hex())
	c.Set(middlewares.SessionId, "current-session")
	c.Set(middlewares.ClientId, "123")

	ChangePassword(useradmin.New(userRepo, sessionRepo, nil, nil))(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.changePasswordCall != userID.Hex() {
		t.Fatalf("expected ChangePassword to be called on self, got %q", userRepo.changePasswordCall)
	}
	if sessionRepo.revokeUserID != userID.Hex() {
		t.Fatalf("expected revoke on self user id, got %q", sessionRepo.revokeUserID)
	}
	if sessionRepo.revokeCurrent != "current-session" {
		t.Fatalf("expected current session to be preserved, got %q", sessionRepo.revokeCurrent)
	}
}

func TestGetUserListUserForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &userUseCaseRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.Role, constant.USER)

	GetUserList(useradmin.New(userRepo, nil, nil, nil))(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.getUsersCalled || userRepo.getUserAllCalledWith != "" {
		t.Fatal("expected no repo call for USER")
	}
}

func TestAddUserByRoleRejectsDuplicateUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)

	existing := &model.User{Id: primitive.NewObjectID(), Username: "bobby", ClientId: "123"}
	userRepo := &userUseCaseRepo{
		byUsername: map[string]*model.User{"bobby": existing},
	}
	systemRepo := &authTestSystemRepo{
		system: &model.System{ClientId: "123", SystemCode: "UM"},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{
		"firstName": "Bob",
		"lastName":  "B",
		"username":  "bobby",
		"password":  "password1",
		"clientId":  "123",
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")
	c.Set(middlewares.UserId, primitive.NewObjectID().Hex())

	AddUserByRole(useradmin.New(userRepo, nil, systemRepo, nil))(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate username, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.createdUser != nil {
		t.Fatal("expected no user creation on duplicate username")
	}
}

func TestGetUserByIdWrongClientNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	targetID := primitive.NewObjectID()
	userRepo := &userUseCaseRepo{
		byId: map[string]*model.User{
			targetID.Hex(): {
				Id:       targetID,
				Username: "bob",
				ClientId: "999",
				Role:     constant.USER,
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: targetID.Hex()}}
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")

	GetUserById(useradmin.New(userRepo, nil, nil, nil))(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 cross-client lookup, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestGetUserInfoReturnsCurrent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	selfID := primitive.NewObjectID()
	userRepo := &userUseCaseRepo{
		byId: map[string]*model.User{
			selfID.Hex(): {
				Id:       selfID,
				Username: "alice",
				ClientId: "123",
				Role:     constant.USER,
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.UserId, selfID.Hex())

	GetUserInfo(userRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp model.User
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Username != "alice" {
		t.Fatalf("expected alice, got %q", resp.Username)
	}
}

func TestUpdateUserInfoUpdatesSelf(t *testing.T) {
	gin.SetMode(gin.TestMode)

	selfID := primitive.NewObjectID()
	userRepo := &userUseCaseRepo{
		byId: map[string]*model.User{
			selfID.Hex(): {Id: selfID, Username: "alice", ClientId: "123", Role: constant.USER},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{
		"firstName": "Alice",
		"lastName":  "A",
	})
	c.Request = httptest.NewRequest(http.MethodPut, "/user/info", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middlewares.UserId, selfID.Hex())
	c.Set(middlewares.ClientId, "123")

	UpdateUserInfo(userRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.updateUserCall != selfID.Hex() {
		t.Fatalf("expected update on self id %s, got %q", selfID.Hex(), userRepo.updateUserCall)
	}
	if userRepo.updateUserClient != "123" {
		t.Fatalf("expected update scoped to client 123, got %q", userRepo.updateUserClient)
	}
}

func TestUserAdminRejectsMalformedBodyWith400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/user/x/role", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "x"}}
	c.Set(middlewares.Role, constant.ADMIN)

	UpdateRoleById(useradmin.New(&userUseCaseRepo{}, nil, nil, nil))(c)

	if w.Code != http.StatusBadRequest || !bytes.Contains(w.Body.Bytes(), []byte(errs.ErrBadRequest)) {
		t.Fatalf("expected 400 %s, got %d body=%s", errs.ErrBadRequest, w.Code, w.Body.String())
	}
}

func TestUserAdminRejectsInvalidIdWith400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "not-hex"}}
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")

	GetUserById(useradmin.New(&invalidHexRepo{}, nil, nil, nil))(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid id, got %d body=%s", w.Code, w.Body.String())
	}
}

type invalidHexRepo struct{ repository.IUser }

func (invalidHexRepo) GetUserByClientId(string, string) (*model.User, error) {
	return nil, primitive.ErrInvalidHex
}
