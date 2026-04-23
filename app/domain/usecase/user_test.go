package usecase

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"um/app/core/constant"
	"um/app/core/utils"
	"um/app/domain/model"
	"um/app/featues/request"
	"um/middlewares"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type userUseCaseRepo struct {
	byId                 map[string]*model.User
	byUsername           map[string]*model.User
	clientIdFilter       map[string]string
	all                  []model.User
	allByClient          map[string][]model.User
	changePasswordCall   string
	setPasswordCall      string
	updateRoleCall       string
	updateRoleValue      string
	updateStatusCall     string
	updateStatusValue    string
	updateUserCall       string
	updateUserClient     string
	createdUser          *model.User
	createdUserRole      string
	removedUserId        string
	removedUserClient    string
	getByClientIdCalls   []string
	getByClientIdScoped  []string
	getUsersCalled       bool
	getUserAllCalledWith string
	updateErr            error
}

func (r *userUseCaseRepo) CreateIndex() (string, error) { return "", nil }
func (r *userUseCaseRepo) GetUsers() ([]model.User, error) {
	r.getUsersCalled = true
	return r.all, nil
}
func (r *userUseCaseRepo) GetUserAll(clientId string) ([]model.User, error) {
	r.getUserAllCalledWith = clientId
	if r.allByClient != nil {
		return r.allByClient[clientId], nil
	}
	return r.all, nil
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
	r.getByClientIdCalls = append(r.getByClientIdCalls, id)
	r.getByClientIdScoped = append(r.getByClientIdScoped, clientId)
	if u, ok := r.byId[id]; ok {
		if clientId != "" && u.ClientId != clientId {
			return nil, mongo.ErrNoDocuments
		}
		return u, nil
	}
	return nil, mongo.ErrNoDocuments
}
func (r *userUseCaseRepo) CreateUser(form request.User, role string) (*model.User, error) {
	r.createdUserRole = role
	u := &model.User{
		Id:       primitive.NewObjectID(),
		Username: utils.NormalizeUsername(form.Username),
		ClientId: form.ClientId,
		Role:     role,
		Status:   constant.ACTIVE,
	}
	r.createdUser = u
	return u, nil
}
func (r *userUseCaseRepo) RemoveUserById(id string, clientId string) (*model.User, error) {
	r.removedUserId = id
	r.removedUserClient = clientId
	if u, ok := r.byId[id]; ok {
		return u, nil
	}
	return nil, mongo.ErrNoDocuments
}
func (r *userUseCaseRepo) UpdateUserById(id string, clientId string, form request.UpdateUser) (*model.User, error) {
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	r.updateUserCall = id
	r.updateUserClient = clientId
	u := r.byId[id]
	return u, nil
}
func (r *userUseCaseRepo) UpdateStatusById(id string, clientId string, form request.UpdateStatus) (*model.User, error) {
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	r.updateStatusCall = id
	r.updateStatusValue = form.Status
	u := r.byId[id]
	return u, nil
}
func (r *userUseCaseRepo) UpdateRoleById(id string, clientId string, form request.UpdateRole) (*model.User, error) {
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	r.updateRoleCall = id
	r.updateRoleValue = form.Role
	u := r.byId[id]
	return u, nil
}
func (r *userUseCaseRepo) ChangePassword(id string, clientId string, form request.ChangePassword) (*model.User, error) {
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	r.changePasswordCall = id
	u := r.byId[id]
	return u, nil
}
func (r *userUseCaseRepo) SetPassword(id string, clientId string, form request.SetPassword) (*model.User, error) {
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	r.setPasswordCall = id
	u := r.byId[id]
	return u, nil
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

	ChangePassword(userRepo, sessionRepo)(c)

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

func TestChangePasswordRejectsWrongOldPasswordWithoutRevoke(t *testing.T) {
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
	sessionRepo := &authTestSessionRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{
		"oldPassword": "wrong-password",
		"newPassword": "new-password-1",
	})
	c.Request = httptest.NewRequest(http.MethodPut, "/user/change-password", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middlewares.UserId, userID.Hex())
	c.Set(middlewares.SessionId, "current-session")
	c.Set(middlewares.ClientId, "123")

	ChangePassword(userRepo, sessionRepo)(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.changePasswordCall != "" {
		t.Fatalf("expected no password change, got %q", userRepo.changePasswordCall)
	}
	if sessionRepo.revokeUserID != "" {
		t.Fatalf("expected no session revoke on wrong password, got %q", sessionRepo.revokeUserID)
	}
}

func TestSetPasswordRevokesAllTargetSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminID := primitive.NewObjectID()
	targetID := primitive.NewObjectID()
	userRepo := &userUseCaseRepo{
		byId: map[string]*model.User{
			targetID.Hex(): {
				Id:       targetID,
				Username: "bob",
				ClientId: "123",
				Role:     constant.USER,
				Status:   constant.ACTIVE,
			},
		},
	}
	sessionRepo := &authTestSessionRepo{revokeCount: 3}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{"password": "brand-new-password"})
	c.Request = httptest.NewRequest(http.MethodPatch, "/user/"+targetID.Hex()+"/set-password", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: targetID.Hex()}}
	c.Set(middlewares.UserId, adminID.Hex())
	c.Set(middlewares.SessionId, "admin-session")
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")

	SetPassword(userRepo, sessionRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.setPasswordCall != targetID.Hex() {
		t.Fatalf("expected SetPassword on target, got %q", userRepo.setPasswordCall)
	}
	if sessionRepo.revokeUserID != targetID.Hex() {
		t.Fatalf("expected revoke on target id, got %q", sessionRepo.revokeUserID)
	}
	if sessionRepo.revokeCurrent != "" {
		t.Fatalf("expected all target sessions revoked (empty current), got %q", sessionRepo.revokeCurrent)
	}
}

func TestUpdateRoleByIdRevokesTargetSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminID := primitive.NewObjectID()
	targetID := primitive.NewObjectID()
	userRepo := &userUseCaseRepo{
		byId: map[string]*model.User{
			targetID.Hex(): {
				Id:       targetID,
				Username: "bob",
				ClientId: "123",
				Role:     constant.MANAGER,
				Status:   constant.ACTIVE,
			},
		},
	}
	sessionRepo := &authTestSessionRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{"role": constant.USER})
	c.Request = httptest.NewRequest(http.MethodPatch, "/user/"+targetID.Hex()+"/role", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: targetID.Hex()}}
	c.Set(middlewares.UserId, adminID.Hex())
	c.Set(middlewares.SessionId, "admin-session")
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")

	UpdateRoleById(userRepo, sessionRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.updateRoleCall != targetID.Hex() || userRepo.updateRoleValue != constant.USER {
		t.Fatalf("expected UpdateRoleById(%s, USER), got (%s, %s)", targetID.Hex(), userRepo.updateRoleCall, userRepo.updateRoleValue)
	}
	if sessionRepo.revokeUserID != targetID.Hex() {
		t.Fatalf("expected revoke on target id, got %q", sessionRepo.revokeUserID)
	}
	if sessionRepo.revokeCurrent != "" {
		t.Fatalf("expected all target sessions revoked (empty current), got %q", sessionRepo.revokeCurrent)
	}
}

func TestUpdateStatusByIdRevokesTargetSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminID := primitive.NewObjectID()
	targetID := primitive.NewObjectID()
	userRepo := &userUseCaseRepo{
		byId: map[string]*model.User{
			targetID.Hex(): {
				Id:       targetID,
				Username: "bob",
				ClientId: "123",
				Role:     constant.USER,
				Status:   constant.ACTIVE,
			},
		},
	}
	sessionRepo := &authTestSessionRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{"status": constant.INACTIVE})
	c.Request = httptest.NewRequest(http.MethodPatch, "/user/"+targetID.Hex()+"/status", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: targetID.Hex()}}
	c.Set(middlewares.UserId, adminID.Hex())
	c.Set(middlewares.SessionId, "admin-session")
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")

	UpdateStatusById(userRepo, sessionRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.updateStatusCall != targetID.Hex() || userRepo.updateStatusValue != constant.INACTIVE {
		t.Fatalf("expected UpdateStatusById(%s, INACTIVE), got (%s, %s)", targetID.Hex(), userRepo.updateStatusCall, userRepo.updateStatusValue)
	}
	if sessionRepo.revokeUserID != targetID.Hex() {
		t.Fatalf("expected revoke on target id, got %q", sessionRepo.revokeUserID)
	}
	if sessionRepo.revokeCurrent != "" {
		t.Fatalf("expected all target sessions revoked (empty current), got %q", sessionRepo.revokeCurrent)
	}
}

func TestUpdateStatusByIdRejectsSelf(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminID := primitive.NewObjectID()
	userRepo := &userUseCaseRepo{
		byId: map[string]*model.User{
			adminID.Hex(): {
				Id:       adminID,
				Username: "admin",
				ClientId: "123",
				Role:     constant.ADMIN,
				Status:   constant.ACTIVE,
			},
		},
	}
	sessionRepo := &authTestSessionRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{"status": constant.INACTIVE})
	c.Request = httptest.NewRequest(http.MethodPatch, "/user/"+adminID.Hex()+"/status", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: adminID.Hex()}}
	c.Set(middlewares.UserId, adminID.Hex())
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")

	UpdateStatusById(userRepo, sessionRepo)(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 self-change rejection, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.updateStatusCall != "" {
		t.Fatalf("expected no status update on self, got %q", userRepo.updateStatusCall)
	}
	if sessionRepo.revokeUserID != "" {
		t.Fatalf("expected no revoke on self, got %q", sessionRepo.revokeUserID)
	}
}

func TestGetUserListSuperReturnsAll(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &userUseCaseRepo{
		all: []model.User{
			{Id: primitive.NewObjectID(), Username: "alice", ClientId: "123"},
			{Id: primitive.NewObjectID(), Username: "bob", ClientId: "456"},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.Role, constant.SUPER)

	GetUserList(userRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !userRepo.getUsersCalled {
		t.Fatal("expected GetUsers to be called for SUPER")
	}
	if userRepo.getUserAllCalledWith != "" {
		t.Fatal("expected GetUserAll NOT to be called for SUPER")
	}
}

func TestGetUserListAdminScopesByClientId(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &userUseCaseRepo{
		allByClient: map[string][]model.User{
			"123": {{Username: "alice", ClientId: "123"}},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")

	GetUserList(userRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.getUserAllCalledWith != "123" {
		t.Fatalf("expected GetUserAll(123), got %q", userRepo.getUserAllCalledWith)
	}
	if userRepo.getUsersCalled {
		t.Fatal("expected ADMIN not to call GetUsers")
	}
}

func TestGetUserListManagerScopesByClientId(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &userUseCaseRepo{
		allByClient: map[string][]model.User{
			"123": {{Username: "alice", ClientId: "123"}},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.Role, constant.MANAGER)
	c.Set(middlewares.ClientId, "123")

	GetUserList(userRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.getUserAllCalledWith != "123" {
		t.Fatalf("expected GetUserAll(123) for MANAGER, got %q", userRepo.getUserAllCalledWith)
	}
}

func TestGetUserListUserForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &userUseCaseRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.Role, constant.USER)

	GetUserList(userRepo)(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.getUsersCalled || userRepo.getUserAllCalledWith != "" {
		t.Fatal("expected no repo call for USER")
	}
}

func TestAddUserByRoleAdminCannotCreateAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &userUseCaseRepo{}
	systemRepo := &authTestSystemRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{
		"firstName": "Bob",
		"lastName":  "B",
		"username":  "bobby",
		"password":  "password1",
		"clientId":  "123",
		"role":      constant.ADMIN,
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")
	c.Set(middlewares.UserId, primitive.NewObjectID().Hex())

	AddUserByRole(userRepo, systemRepo)(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.createdUser != nil {
		t.Fatal("expected no user creation when ADMIN tries to create ADMIN")
	}
}

func TestAddUserByRoleAdminCreatesUserInOwnClient(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &userUseCaseRepo{}
	systemRepo := &authTestSystemRepo{
		system: &model.System{
			Id:         primitive.NewObjectID(),
			ClientId:   "123",
			SystemCode: "UM",
		},
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

	AddUserByRole(userRepo, systemRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.createdUserRole != constant.USER {
		t.Fatalf("expected default ADMIN create → USER, got %q", userRepo.createdUserRole)
	}
}

func TestAddUserByRoleAdminRejectsOtherClient(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &userUseCaseRepo{}
	systemRepo := &authTestSystemRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{
		"firstName": "Bob",
		"lastName":  "B",
		"username":  "bobby",
		"password":  "password1",
		"clientId":  "999",
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")
	c.Set(middlewares.UserId, primitive.NewObjectID().Hex())

	AddUserByRole(userRepo, systemRepo)(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for ADMIN creating in other client, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.createdUser != nil {
		t.Fatal("expected no user creation across clients")
	}
}

func TestAddUserByRoleSuperRejectsSuperOutsideBootstrapClient(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &userUseCaseRepo{}
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
		"role":      constant.SUPER,
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/user", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middlewares.Role, constant.SUPER)
	c.Set(middlewares.UserId, primitive.NewObjectID().Hex())

	AddUserByRole(userRepo, systemRepo)(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for SUPER outside clientId=000, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.createdUser != nil {
		t.Fatal("expected no user creation")
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

	AddUserByRole(userRepo, systemRepo)(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate username, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.createdUser != nil {
		t.Fatal("expected no user creation on duplicate username")
	}
}

func TestDeleteUserByIdRejectsSelf(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminID := primitive.NewObjectID()
	userRepo := &userUseCaseRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: adminID.Hex()}}
	c.Set(middlewares.UserId, adminID.Hex())
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")

	DeleteUserById(userRepo)(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 self-delete rejection, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.removedUserId != "" {
		t.Fatalf("expected no removal on self-delete, got %q", userRepo.removedUserId)
	}
}

func TestDeleteUserByIdAdminCannotDeleteAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminID := primitive.NewObjectID()
	targetID := primitive.NewObjectID()
	userRepo := &userUseCaseRepo{
		byId: map[string]*model.User{
			targetID.Hex(): {
				Id:       targetID,
				Username: "bob",
				ClientId: "123",
				Role:     constant.ADMIN,
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: targetID.Hex()}}
	c.Set(middlewares.UserId, adminID.Hex())
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")

	DeleteUserById(userRepo)(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 ADMIN deleting ADMIN, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.removedUserId != "" {
		t.Fatal("expected no removal when role validation fails")
	}
}

func TestDeleteUserByIdAdminDeletesUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminID := primitive.NewObjectID()
	targetID := primitive.NewObjectID()
	userRepo := &userUseCaseRepo{
		byId: map[string]*model.User{
			targetID.Hex(): {
				Id:       targetID,
				Username: "bob",
				ClientId: "123",
				Role:     constant.USER,
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: targetID.Hex()}}
	c.Set(middlewares.UserId, adminID.Hex())
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")

	DeleteUserById(userRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.removedUserId != targetID.Hex() {
		t.Fatalf("expected removal of %s, got %q", targetID.Hex(), userRepo.removedUserId)
	}
	if userRepo.removedUserClient != "123" {
		t.Fatalf("expected client-scoped remove %q, got %q", "123", userRepo.removedUserClient)
	}
}

func TestGetUserByIdAdminScopedByClientId(t *testing.T) {
	gin.SetMode(gin.TestMode)

	targetID := primitive.NewObjectID()
	userRepo := &userUseCaseRepo{
		byId: map[string]*model.User{
			targetID.Hex(): {
				Id:       targetID,
				Username: "bob",
				ClientId: "123",
				Role:     constant.USER,
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: targetID.Hex()}}
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")

	GetUserById(userRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if len(userRepo.getByClientIdScoped) != 1 || userRepo.getByClientIdScoped[0] != "123" {
		t.Fatalf("expected lookup scoped to client 123, got %#v", userRepo.getByClientIdScoped)
	}
}

func TestGetUserByIdSuperUnscoped(t *testing.T) {
	gin.SetMode(gin.TestMode)

	targetID := primitive.NewObjectID()
	userRepo := &userUseCaseRepo{
		byId: map[string]*model.User{
			targetID.Hex(): {
				Id:       targetID,
				Username: "bob",
				ClientId: "456",
				Role:     constant.USER,
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: targetID.Hex()}}
	c.Set(middlewares.Role, constant.SUPER)

	GetUserById(userRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for SUPER cross-client read, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.getByClientIdScoped[0] != "" {
		t.Fatalf("SUPER should query unscoped, got %q", userRepo.getByClientIdScoped[0])
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

	GetUserById(userRepo)(c)

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

func TestUpdateUserByIdSelfPathSkipsRoleValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	selfID := primitive.NewObjectID()
	userRepo := &userUseCaseRepo{
		byId: map[string]*model.User{
			selfID.Hex(): {Id: selfID, Username: "alice", ClientId: "123", Role: constant.ADMIN},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{
		"firstName": "Alice",
		"lastName":  "A",
	})
	c.Request = httptest.NewRequest(http.MethodPut, "/user/"+selfID.Hex(), bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: selfID.Hex()}}
	c.Set(middlewares.UserId, selfID.Hex())
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")

	UpdateUserById(userRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 self-update, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.updateUserCall != selfID.Hex() {
		t.Fatalf("expected update on self, got %q", userRepo.updateUserCall)
	}
}

func TestUpdateUserByIdForbidsAdminUpdatingAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminID := primitive.NewObjectID()
	targetID := primitive.NewObjectID()
	userRepo := &userUseCaseRepo{
		byId: map[string]*model.User{
			targetID.Hex(): {Id: targetID, Username: "bob", ClientId: "123", Role: constant.ADMIN},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{
		"firstName": "Bob",
		"lastName":  "B",
	})
	c.Request = httptest.NewRequest(http.MethodPut, "/user/"+targetID.Hex(), bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: targetID.Hex()}}
	c.Set(middlewares.UserId, adminID.Hex())
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")

	UpdateUserById(userRepo)(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
	if userRepo.updateUserCall != "" {
		t.Fatalf("expected no update, got %q", userRepo.updateUserCall)
	}
}

type unlockLoginGuard struct {
	unlocked []string
	err      error
}

func (g *unlockLoginGuard) IsLocked(username string) (bool, error) { return false, nil }
func (g *unlockLoginGuard) RecordFailure(username string) error    { return nil }
func (g *unlockLoginGuard) Reset(username string) error            { return nil }
func (g *unlockLoginGuard) Unlock(username string) error {
	g.unlocked = append(g.unlocked, username)
	return g.err
}

func TestUnlockUserByIdCallsLoginGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)

	targetID := primitive.NewObjectID()
	userRepo := &userUseCaseRepo{
		byId: map[string]*model.User{
			targetID.Hex(): {Id: targetID, Username: "bob", ClientId: "123", Role: constant.USER},
		},
	}
	guard := &unlockLoginGuard{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: targetID.Hex()}}
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")

	UnlockUserById(userRepo, guard)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if len(guard.unlocked) != 1 || guard.unlocked[0] != "bob" {
		t.Fatalf("expected Unlock('bob'), got %#v", guard.unlocked)
	}
}

func TestUnlockUserByIdRejectsNonManageable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	targetID := primitive.NewObjectID()
	userRepo := &userUseCaseRepo{
		byId: map[string]*model.User{
			targetID.Hex(): {Id: targetID, Username: "bob", ClientId: "123", Role: constant.ADMIN},
		},
	}
	guard := &unlockLoginGuard{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: targetID.Hex()}}
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")

	UnlockUserById(userRepo, guard)(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for ADMIN unlocking another ADMIN, got %d body=%s", w.Code, w.Body.String())
	}
	if len(guard.unlocked) != 0 {
		t.Fatalf("expected no unlock on role rejection, got %#v", guard.unlocked)
	}
}
