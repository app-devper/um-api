package usecase

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"um/app/core/constant"
	"um/app/domain/model"
	"um/app/domain/repository"
	"um/middlewares"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCreateSSOTicketReturnsTicketForActiveUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

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
	ssoRepo := &authTestSSOTicketRepo{createRes: "ticket-abc"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.UserId, userID.Hex())
	c.Set(middlewares.Role, constant.ADMIN)
	c.Set(middlewares.ClientId, "123")
	c.Set(middlewares.System, "UM")

	CreateSSOTicket(ssoRepo, userRepo)(c)

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

func TestCreateSSOTicketRejectsInactiveUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := primitive.NewObjectID()
	userRepo := &authTestUserRepo{
		user: &model.User{
			Id:       userID,
			Username: "alice",
			ClientId: "123",
			Role:     constant.ADMIN,
			Status:   constant.INACTIVE,
		},
	}
	ssoRepo := &authTestSSOTicketRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.UserId, userID.Hex())

	CreateSSOTicket(ssoRepo, userRepo)(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for inactive user, got %d body=%s", w.Code, w.Body.String())
	}
	if ssoRepo.createInput != nil {
		t.Fatal("expected no ticket creation for inactive user")
	}
}

func TestCreateSSOTicketUserNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &authTestUserRepo{}
	ssoRepo := &authTestSSOTicketRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.UserId, primitive.NewObjectID().Hex())

	CreateSSOTicket(ssoRepo, userRepo)(c)

	if w.Code != http.StatusInternalServerError && w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 or 500 for missing user, got %d body=%s", w.Code, w.Body.String())
	}
	if ssoRepo.createInput != nil {
		t.Fatal("expected no ticket creation when user missing")
	}
}

func TestExchangeSSOTicketActiveUserMintsFreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SECRET_KEY", "test-secret")

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

	ExchangeSSOTicket(ssoRepo, userRepo, sessionRepo)(c)

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
	t.Setenv("SECRET_KEY", "test-secret")

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

	ExchangeSSOTicket(ssoRepo, userRepo, sessionRepo)(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for inactive user, got %d body=%s", w.Code, w.Body.String())
	}
	if sessionRepo.created {
		t.Fatal("expected no session for inactive user")
	}
}
