package usecase

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"um/app/core/constant"
	"um/app/core/errs"
	"um/app/domain/model"
	"um/app/domain/repository"
	"um/app/domain/session"
	"um/middlewares"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCreateSSOTicketReturnsTicketForActiveUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := primitive.NewObjectID()
	ssoRepo := &authTestSSOTicketRepo{createRes: "ticket-abc"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.UserId, userID.Hex())
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")
	c.Set(middlewares.System, "UM")

	c.Request = httptest.NewRequest(http.MethodPost, "/auth/sso-ticket", nil)

	CreateSSOTicket(ssoManager("test-secret", &authTestSessionRepo{}, &authTestUserRepo{}), ssoRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Ticket    string `json:"ticket"`
		ExpiresIn int    `json:"expiresIn"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Ticket != "ticket-abc" {
		t.Fatalf("expected ticket-abc, got %q", resp.Ticket)
	}
	if resp.ExpiresIn != int(repository.SSOTicketTTL.Seconds()) {
		t.Fatalf("expected expiresIn=%d, got %d", int(repository.SSOTicketTTL.Seconds()), resp.ExpiresIn)
	}
	if ssoRepo.createInput == nil {
		t.Fatal("expected Create to be called")
	}
	if ssoRepo.createInput.UserId != userID.Hex() {
		t.Fatalf("expected payload userId=%s, got %q", userID.Hex(), ssoRepo.createInput.UserId)
	}
	if ssoRepo.createInput.Role != constant.ADMIN || ssoRepo.createInput.ClientId != "123" || ssoRepo.createInput.System != "UM" {
		t.Fatalf("unexpected ticket payload: %+v", ssoRepo.createInput)
	}
}

func TestExchangeSSOTicketActiveUserMintsFreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := primitive.NewObjectID()
	userRepo := &authTestUserRepo{
		user: &model.User{
			Id:       userID,
			Username: "alice",
			ClientId: "123",
			Role:     constant.MANAGER,
			Status:   constant.ACTIVE,
		},
	}
	sessionRepo := &authTestSessionRepo{}
	ssoRepo := &authTestSSOTicketRepo{
		payload: &repository.TicketPayload{
			UserId:   userID.Hex(),
			Role:     constant.USER,
			ClientId: "STALE",
			System:   "UM",
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := []byte(`{"ticket":"some-ticket"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/exchange", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	ExchangeSSOTicket(ssoManager("test-secret", sessionRepo, userRepo), ssoRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !sessionRepo.created {
		t.Fatal("expected new session to be created")
	}
	var resp struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.AccessToken == "" {
		t.Fatal("expected access token")
	}
}

func TestExchangeSSOTicketRejectsInactiveUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := primitive.NewObjectID()
	userRepo := &authTestUserRepo{
		user: &model.User{
			Id:       userID,
			Username: "alice",
			ClientId: "123",
			Role:     constant.USER,
			Status:   constant.INACTIVE,
		},
	}
	sessionRepo := &authTestSessionRepo{}
	ssoRepo := &authTestSSOTicketRepo{
		payload: &repository.TicketPayload{
			UserId:   userID.Hex(),
			Role:     constant.USER,
			ClientId: "123",
			System:   "UM",
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := []byte(`{"ticket":"some-ticket"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/exchange", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	ExchangeSSOTicket(ssoManager("test-secret", sessionRepo, userRepo), ssoRepo)(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for inactive user, got %d body=%s", w.Code, w.Body.String())
	}
	if sessionRepo.created {
		t.Fatal("expected no session for inactive user")
	}
}

// ssoManager is a session Manager whose Client "123" has Systems UM and POS.
func ssoManager(secret string, sessions repository.ISession, users repository.IUser) *session.Manager {
	systems := &authTestSystemRepo{byClient: map[string][]model.System{"123": {
		{ClientId: "123", SystemCode: "UM", SystemName: "User Management", Host: "https://api.devper.app", WebUrl: "https://um.devper.app"},
		{ClientId: "123", SystemCode: "POS", SystemName: "Point of Sale", Host: "https://pos-api.devper.app", WebUrl: "https://pos.devper.app/"},
	}}}
	return session.NewManager(secret, sessions, users, session.WithLogin(systems, nil))
}

func ticketRequest(t *testing.T, user *model.User, body string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/sso-ticket", bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middlewares.UserId, user.Id.Hex())
	c.Set(middlewares.Role, user.Role)
	c.Set(middlewares.ClientId, user.ClientId)
	c.Set(middlewares.System, "UM")
	return c, w
}

func TestCreateSSOTicketForAnotherSystemNamesThatSystem(t *testing.T) {
	user := &model.User{Id: primitive.NewObjectID(), ClientId: "123", Role: constant.USER, Status: constant.ACTIVE}
	ssoRepo := &authTestSSOTicketRepo{createRes: "ticket-pos"}
	c, w := ticketRequest(t, user, `{"system":"POS"}`)

	CreateSSOTicket(ssoManager("test-secret", &authTestSessionRepo{}, &authTestUserRepo{user: user}), ssoRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if ssoRepo.createInput == nil || ssoRepo.createInput.System != "POS" {
		t.Fatalf("expected a ticket for POS, got %+v", ssoRepo.createInput)
	}
}

func TestCreateSSOTicketRefusesASystemTheUserMayNotUse(t *testing.T) {
	user := &model.User{Id: primitive.NewObjectID(), ClientId: "123", Role: constant.USER, Status: constant.ACTIVE}
	ssoRepo := &authTestSSOTicketRepo{createRes: "ticket-gold"}
	c, w := ticketRequest(t, user, `{"system":"GOLD"}`)

	CreateSSOTicket(ssoManager("test-secret", &authTestSessionRepo{}, &authTestUserRepo{user: user}), ssoRepo)(c)

	if w.Code != http.StatusForbidden || !bytes.Contains(w.Body.Bytes(), []byte(errs.ErrNoPermission)) {
		t.Fatalf("expected 403 %s, got %d body=%s", errs.ErrNoPermission, w.Code, w.Body.String())
	}
	if ssoRepo.createInput != nil {
		t.Fatalf("expected no ticket, got %+v", ssoRepo.createInput)
	}
}

func TestLaunchableSystemsListsTheOtherSystemsWithTheirOrigin(t *testing.T) {
	user := &model.User{Id: primitive.NewObjectID(), ClientId: "123", Role: constant.USER, Status: constant.ACTIVE}
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/systems", nil)
	c.Set(middlewares.UserId, user.Id.Hex())
	c.Set(middlewares.System, "UM")

	LaunchableSystems(ssoManager("test-secret", &authTestSessionRepo{}, &authTestUserRepo{user: user}))(c)

	want := `[{"systemCode":"POS","systemName":"Point of Sale","webUrl":"https://pos.devper.app"}]`
	if w.Code != http.StatusOK || w.Body.String() != want {
		t.Fatalf("got %d %s, want %s", w.Code, w.Body.String(), want)
	}
}
