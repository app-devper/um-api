package usecase

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"um/app/core/config"
	"um/app/core/constant"
	"um/app/core/utils"
	"um/app/domain/model"
	"um/app/domain/repository"
	"um/app/featues/request"
	"um/middlewares"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type authTestUserRepo struct {
	user *model.User
}

func (r *authTestUserRepo) CreateIndex() (string, error) { return "", nil }
func (r *authTestUserRepo) GetUsers() ([]model.User, error) {
	panic("unexpected call")
}
func (r *authTestUserRepo) GetUserAll(clientId string) ([]model.User, error) {
	panic("unexpected call")
}
func (r *authTestUserRepo) GetUserByUsername(username string) (*model.User, error) {
	if r.user == nil || r.user.Username != utils.NormalizeUsername(username) {
		return nil, errors.New("not found")
	}
	return r.user, nil
}
func (r *authTestUserRepo) GetUserById(id string) (*model.User, error) {
	if r.user == nil || r.user.Id.Hex() != id {
		return nil, errors.New("not found")
	}
	return r.user, nil
}
func (r *authTestUserRepo) GetUserByClientId(id string, clientId string) (*model.User, error) {
	panic("unexpected call")
}
func (r *authTestUserRepo) CreateUser(form request.User, role string) (*model.User, error) {
	panic("unexpected call")
}
func (r *authTestUserRepo) RemoveUserById(id string, clientId string) (*model.User, error) {
	panic("unexpected call")
}
func (r *authTestUserRepo) UpdateUserById(id string, clientId string, form request.UpdateUser) (*model.User, error) {
	panic("unexpected call")
}
func (r *authTestUserRepo) UpdateStatusById(id string, clientId string, form request.UpdateStatus) (*model.User, error) {
	panic("unexpected call")
}
func (r *authTestUserRepo) UpdateRoleById(id string, clientId string, form request.UpdateRole) (*model.User, error) {
	panic("unexpected call")
}
func (r *authTestUserRepo) ChangePassword(id string, clientId string, form request.ChangePassword) (*model.User, error) {
	panic("unexpected call")
}
func (r *authTestUserRepo) SetPassword(id string, clientId string, form request.SetPassword) (*model.User, error) {
	panic("unexpected call")
}

type authTestSessionRepo struct {
	created        bool
	system         string
	removed        []string
	getByID        map[string]string
	listItems      []repository.SessionInfo
	listUserID     string
	listCurrentID  string
	revokeUserID   string
	revokeCurrent  string
	revokeCount    int
	updatedSession string
	updatedTTL     time.Duration
	getErr         error
	removeErr      error
	listErr        error
	revokeOtherErr error
	updateErr      error
}

func (r *authTestSessionRepo) CreateSession(userId string, expiration time.Duration, metadata repository.SessionMetadata) (string, error) {
	r.created = true
	r.system = metadata.System
	return "session-123", nil
}
func (r *authTestSessionRepo) UpdateSessionExpireById(sessionId string, expiration time.Duration) error {
	r.updatedSession = sessionId
	r.updatedTTL = expiration
	if r.updateErr != nil {
		return r.updateErr
	}
	return nil
}
func (r *authTestSessionRepo) RemoveSessionById(sessionId string) error {
	if r.removeErr != nil {
		return r.removeErr
	}
	r.removed = append(r.removed, sessionId)
	return nil
}
func (r *authTestSessionRepo) GetSessionById(sessionId string) (string, error) {
	if r.getErr != nil {
		return "", r.getErr
	}
	userID, ok := r.getByID[sessionId]
	if !ok {
		return "", errors.New("not found")
	}
	return userID, nil
}
func (r *authTestSessionRepo) ListUserSessions(userId string, currentSessionId string) ([]repository.SessionInfo, error) {
	r.listUserID = userId
	r.listCurrentID = currentSessionId
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.listItems, nil
}
func (r *authTestSessionRepo) RevokeOtherSessions(userId string, currentSessionId string) (int, error) {
	r.revokeUserID = userId
	r.revokeCurrent = currentSessionId
	if r.revokeOtherErr != nil {
		return 0, r.revokeOtherErr
	}
	return r.revokeCount, nil
}

type authTestSystemRepo struct {
	system *model.System
	err    error
}

func (r *authTestSystemRepo) GetSystems(form request.GetSystems) ([]model.System, error) {
	panic("unexpected call")
}
func (r *authTestSystemRepo) GetSystemsByClientId(clientId string) ([]model.System, error) {
	panic("unexpected call")
}
func (r *authTestSystemRepo) GetSystemsByCode(systemCode string) ([]model.System, error) {
	panic("unexpected call")
}
func (r *authTestSystemRepo) GetSystemById(id string) (*model.System, error) {
	panic("unexpected call")
}
func (r *authTestSystemRepo) GetSystem(clientId string, systemCode string) (*model.System, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.system != nil && r.system.ClientId == clientId && r.system.SystemCode == systemCode {
		return r.system, nil
	}
	return nil, errors.New("not found")
}
func (r *authTestSystemRepo) CreateSystem(form request.System) (*model.System, error) {
	panic("unexpected call")
}
func (r *authTestSystemRepo) RemoveSystemById(id string) (*model.System, error) {
	panic("unexpected call")
}
func (r *authTestSystemRepo) UpdateSystemById(id string, form request.UpdateSystem) (*model.System, error) {
	panic("unexpected call")
}

type authTestLoginGuard struct {
	recorded []string
	reset    []string
}

func (g *authTestLoginGuard) IsLocked(username string) (bool, error) { return false, nil }
func (g *authTestLoginGuard) RecordFailure(username string) error {
	g.recorded = append(g.recorded, username)
	return nil
}
func (g *authTestLoginGuard) Reset(username string) error {
	g.reset = append(g.reset, username)
	return nil
}
func (g *authTestLoginGuard) Unlock(username string) error { panic("unexpected call") }

type authTestSSOTicketRepo struct {
	payload *repository.TicketPayload
	err     error
}

func (r *authTestSSOTicketRepo) Create(payload repository.TicketPayload) (string, error) {
	panic("unexpected call")
}

func (r *authTestSSOTicketRepo) Consume(ticket string) (*repository.TicketPayload, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.payload == nil {
		return nil, errors.New("not found")
	}
	return r.payload, nil
}

func TestLoginRejectsUnknownSystemForUserTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SECRET_KEY", "test-secret")

	hashed, err := utils.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	userRepo := &authTestUserRepo{
		user: &model.User{
			Id:       primitive.NewObjectID(),
			Username: "alice",
			Password: hashed,
			ClientId: "123",
			Role:     constant.ADMIN,
			Status:   constant.ACTIVE,
		},
	}
	sessionRepo := &authTestSessionRepo{}
	systemRepo := &authTestSystemRepo{}
	loginGuard := &authTestLoginGuard{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{
		"username": " alice ",
		"password": "password123",
		"system":   "UM",
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	Login(userRepo, sessionRepo, systemRepo, loginGuard)(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
	if sessionRepo.created {
		t.Fatal("expected no session to be created when system is invalid")
	}
	if len(loginGuard.recorded) != 1 || loginGuard.recorded[0] != "alice" {
		t.Fatalf("expected normalized username to be recorded as failure, got %#v", loginGuard.recorded)
	}
}

func TestLoginCreatesSessionWhenSystemBelongsToUserTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SECRET_KEY", "test-secret")

	hashed, err := utils.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	userRepo := &authTestUserRepo{
		user: &model.User{
			Id:       primitive.NewObjectID(),
			Username: "alice",
			Password: hashed,
			ClientId: "123",
			Role:     constant.ADMIN,
			Status:   constant.ACTIVE,
		},
	}
	sessionRepo := &authTestSessionRepo{}
	systemRepo := &authTestSystemRepo{
		system: &model.System{
			Id:         primitive.NewObjectID(),
			ClientId:   "123",
			SystemCode: "UM",
		},
	}
	loginGuard := &authTestLoginGuard{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{
		"username": " alice ",
		"password": "password123",
		"system":   "UM",
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	Login(userRepo, sessionRepo, systemRepo, loginGuard)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !sessionRepo.created {
		t.Fatal("expected session to be created")
	}
	if sessionRepo.system != "UM" {
		t.Fatalf("expected session system UM, got %q", sessionRepo.system)
	}
	if len(loginGuard.recorded) != 0 {
		t.Fatalf("expected no recorded failures, got %#v", loginGuard.recorded)
	}
	if len(loginGuard.reset) != 1 || loginGuard.reset[0] != "alice" {
		t.Fatalf("expected normalized username reset, got %#v", loginGuard.reset)
	}
	var resp struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.AccessToken == "" {
		t.Fatal("expected access token in response")
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SECRET_KEY", "test-secret")

	hashed, err := utils.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	userRepo := &authTestUserRepo{
		user: &model.User{
			Id:       primitive.NewObjectID(),
			Username: "alice",
			Password: hashed,
			ClientId: "123",
			Role:     constant.ADMIN,
			Status:   constant.ACTIVE,
		},
	}
	sessionRepo := &authTestSessionRepo{}
	systemRepo := &authTestSystemRepo{
		system: &model.System{
			Id:         primitive.NewObjectID(),
			ClientId:   "123",
			SystemCode: "UM",
		},
	}
	loginGuard := &authTestLoginGuard{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{
		"username": " alice ",
		"password": "wrong-password",
		"system":   "UM",
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	Login(userRepo, sessionRepo, systemRepo, loginGuard)(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
	if sessionRepo.created {
		t.Fatal("expected no session to be created")
	}
	if len(loginGuard.recorded) != 1 || loginGuard.recorded[0] != "alice" {
		t.Fatalf("expected normalized username to be recorded as failure, got %#v", loginGuard.recorded)
	}
	if len(loginGuard.reset) != 0 {
		t.Fatalf("expected no reset on wrong password, got %#v", loginGuard.reset)
	}
}

func TestLoginRejectsInactiveUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SECRET_KEY", "test-secret")

	hashed, err := utils.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	userRepo := &authTestUserRepo{
		user: &model.User{
			Id:       primitive.NewObjectID(),
			Username: "alice",
			Password: hashed,
			ClientId: "123",
			Role:     constant.ADMIN,
			Status:   constant.INACTIVE,
		},
	}
	sessionRepo := &authTestSessionRepo{}
	systemRepo := &authTestSystemRepo{
		system: &model.System{
			Id:         primitive.NewObjectID(),
			ClientId:   "123",
			SystemCode: "UM",
		},
	}
	loginGuard := &authTestLoginGuard{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{
		"username": " alice ",
		"password": "password123",
		"system":   "UM",
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	Login(userRepo, sessionRepo, systemRepo, loginGuard)(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
	if sessionRepo.created {
		t.Fatal("expected no session to be created")
	}
	if len(loginGuard.recorded) != 1 || loginGuard.recorded[0] != "alice" {
		t.Fatalf("expected normalized username to be recorded for inactive user, got %#v", loginGuard.recorded)
	}
	if len(loginGuard.reset) != 0 {
		t.Fatalf("expected no reset for inactive user, got %#v", loginGuard.reset)
	}
}

func TestLoginRemovesSessionWhenTokenGenerationFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SECRET_KEY", "")

	hashed, err := utils.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	userRepo := &authTestUserRepo{
		user: &model.User{
			Id:       primitive.NewObjectID(),
			Username: "alice",
			Password: hashed,
			ClientId: "123",
			Role:     constant.ADMIN,
			Status:   constant.ACTIVE,
		},
	}
	sessionRepo := &authTestSessionRepo{}
	systemRepo := &authTestSystemRepo{
		system: &model.System{
			Id:         primitive.NewObjectID(),
			ClientId:   "123",
			SystemCode: "UM",
		},
	}
	loginGuard := &authTestLoginGuard{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{
		"username": "alice",
		"password": "password123",
		"system":   "UM",
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	Login(userRepo, sessionRepo, systemRepo, loginGuard)(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}
	if !sessionRepo.created {
		t.Fatal("expected session to be created before token generation")
	}
	if len(sessionRepo.removed) != 1 || sessionRepo.removed[0] != "session-123" {
		t.Fatalf("expected leaked session to be removed, got %#v", sessionRepo.removed)
	}
}

func TestExchangeSSOTicketRemovesSessionWhenTokenGenerationFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SECRET_KEY", "")

	userID := primitive.NewObjectID()
	userRepo := &authTestUserRepo{
		user: &model.User{
			Id:       userID,
			Username: "alice",
			ClientId: "123",
			Role:     constant.ADMIN,
			Status:   constant.ACTIVE,
		},
	}
	sessionRepo := &authTestSessionRepo{}
	ssoRepo := &authTestSSOTicketRepo{
		payload: &repository.TicketPayload{
			UserId:   userID.Hex(),
			Role:     constant.ADMIN,
			ClientId: "123",
			System:   "UM",
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{
		"ticket": "ticket-123",
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/exchange", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	ExchangeSSOTicket(ssoRepo, userRepo, sessionRepo)(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}
	if !sessionRepo.created {
		t.Fatal("expected session to be created before token generation")
	}
	if len(sessionRepo.removed) != 1 || sessionRepo.removed[0] != "session-123" {
		t.Fatalf("expected leaked session to be removed, got %#v", sessionRepo.removed)
	}
}

func TestRequireSessionSetsUserIDOnSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sessionRepo := &authTestSessionRepo{
		getByID: map[string]string{
			"session-123": "user-123",
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.SessionId, "session-123")

	called := false
	handler := RequireSession(sessionRepo)
	handler(c)
	if !c.IsAborted() {
		called = true
	}

	if !called {
		t.Fatal("expected middleware to continue on valid session")
	}
	if got := c.GetString(middlewares.UserId); got != "user-123" {
		t.Fatalf("expected user id to be set, got %q", got)
	}
}

func TestRequireSessionRejectsInvalidSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sessionRepo := &authTestSessionRepo{getErr: errors.New("redis down")}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.SessionId, "session-123")

	RequireSession(sessionRepo)(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
	if got := c.GetString(middlewares.UserId); got != "" {
		t.Fatalf("expected no user id to be set, got %q", got)
	}
}

func TestListSessionsReturnsItems(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sessionRepo := &authTestSessionRepo{
		listItems: []repository.SessionInfo{
			{SessionId: "s1", Current: true},
			{SessionId: "s2", Current: false},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.UserId, "user-123")
	c.Set(middlewares.SessionId, "s1")

	ListSessions(sessionRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if sessionRepo.listUserID != "user-123" || sessionRepo.listCurrentID != "s1" {
		t.Fatalf("expected list inputs to be forwarded, got user=%q current=%q", sessionRepo.listUserID, sessionRepo.listCurrentID)
	}
	var items []repository.SessionInfo
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(items))
	}
}

func TestRevokeSessionRejectsCurrentSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sessionRepo := &authTestSessionRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "current-session"}}
	c.Set(middlewares.UserId, "user-123")
	c.Set(middlewares.SessionId, "current-session")

	RevokeSession(sessionRepo)(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	if len(sessionRepo.removed) != 0 {
		t.Fatalf("expected no session removal, got %#v", sessionRepo.removed)
	}
}

func TestRevokeSessionRejectsOtherUsersSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sessionRepo := &authTestSessionRepo{
		getByID: map[string]string{
			"target-session": "other-user",
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "target-session"}}
	c.Set(middlewares.UserId, "user-123")
	c.Set(middlewares.SessionId, "current-session")

	RevokeSession(sessionRepo)(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
	if len(sessionRepo.removed) != 0 {
		t.Fatalf("expected no session removal, got %#v", sessionRepo.removed)
	}
}

func TestRevokeSessionRemovesOwnedSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sessionRepo := &authTestSessionRepo{
		getByID: map[string]string{
			"target-session": "user-123",
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "target-session"}}
	c.Set(middlewares.UserId, "user-123")
	c.Set(middlewares.SessionId, "current-session")

	RevokeSession(sessionRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if len(sessionRepo.removed) != 1 || sessionRepo.removed[0] != "target-session" {
		t.Fatalf("expected target session to be removed, got %#v", sessionRepo.removed)
	}
}

func TestRevokeOtherSessionsReturnsCount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sessionRepo := &authTestSessionRepo{revokeCount: 3}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.UserId, "user-123")
	c.Set(middlewares.SessionId, "current-session")

	RevokeOtherSessions(sessionRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if sessionRepo.revokeUserID != "user-123" || sessionRepo.revokeCurrent != "current-session" {
		t.Fatalf("expected revoke inputs to be forwarded, got user=%q current=%q", sessionRepo.revokeUserID, sessionRepo.revokeCurrent)
	}
	var resp struct {
		Message string `json:"message"`
		Revoked int    `json:"revoked"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Revoked != 3 {
		t.Fatalf("expected revoked count 3, got %d", resp.Revoked)
	}
}

func TestKeepAliveReturnsTokenAndExtendsSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SECRET_KEY", "test-secret")

	userID := primitive.NewObjectID()
	userRepo := &authTestUserRepo{
		user: &model.User{
			Id:       userID,
			Username: "alice",
			ClientId: "123",
			Role:     constant.ADMIN,
			Status:   constant.ACTIVE,
		},
	}
	sessionRepo := &authTestSessionRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.SessionId, "session-123")
	c.Set(middlewares.UserId, userID.Hex())
	c.Set(middlewares.System, "UM")

	KeepAlive(userRepo, sessionRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if sessionRepo.updatedSession != "session-123" {
		t.Fatalf("expected keep-alive to refresh session-123, got %q", sessionRepo.updatedSession)
	}
	if sessionRepo.updatedTTL != config.AccessTokenTime {
		t.Fatalf("expected keep-alive ttl %v, got %v", config.AccessTokenTime, sessionRepo.updatedTTL)
	}
	var resp struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.AccessToken == "" {
		t.Fatal("expected access token")
	}
}

func TestKeepAliveRemovesSessionWhenTokenGenerationFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SECRET_KEY", "")

	userID := primitive.NewObjectID()
	userRepo := &authTestUserRepo{
		user: &model.User{
			Id:       userID,
			Username: "alice",
			ClientId: "123",
			Role:     constant.ADMIN,
			Status:   constant.ACTIVE,
		},
	}
	sessionRepo := &authTestSessionRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.SessionId, "session-123")
	c.Set(middlewares.UserId, userID.Hex())
	c.Set(middlewares.System, "UM")

	KeepAlive(userRepo, sessionRepo)(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}
	if len(sessionRepo.removed) != 1 || sessionRepo.removed[0] != "session-123" {
		t.Fatalf("expected current session to be removed on signing failure, got %#v", sessionRepo.removed)
	}
}

func TestLogoutRemovesSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sessionRepo := &authTestSessionRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.SessionId, "session-123")

	Logout(sessionRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if len(sessionRepo.removed) != 1 || sessionRepo.removed[0] != "session-123" {
		t.Fatalf("expected session to be removed, got %#v", sessionRepo.removed)
	}
}

func TestVerifyPasswordRejectsWrongPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hashed, err := utils.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	userID := primitive.NewObjectID()
	userRepo := &authTestUserRepo{
		user: &model.User{
			Id:       userID,
			Username: "alice",
			Password: hashed,
			ClientId: "123",
			Role:     constant.ADMIN,
			Status:   constant.ACTIVE,
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{
		"password":  "wrong-password",
		"objective": "sensitive-action",
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/verify-password", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middlewares.UserId, userID.Hex())

	VerifyPassword(userRepo)(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestVerifyPasswordReturnsSuccessForCorrectPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hashed, err := utils.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	userID := primitive.NewObjectID()
	userRepo := &authTestUserRepo{
		user: &model.User{
			Id:       userID,
			Username: "alice",
			Password: hashed,
			ClientId: "123",
			Role:     constant.ADMIN,
			Status:   constant.ACTIVE,
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{
		"password":  "password123",
		"objective": "sensitive-action",
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/verify-password", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middlewares.UserId, userID.Hex())

	VerifyPassword(userRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Message != "success" {
		t.Fatalf("expected success message, got %q", resp.Message)
	}
}
