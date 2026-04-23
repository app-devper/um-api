package usecase

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"um/app/domain/model"
	"um/middlewares"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestGetSystemReturnsSystemForClient(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sysRepo := &authTestSystemRepo{
		system: &model.System{
			Id:         primitive.NewObjectID(),
			ClientId:   "123",
			SystemCode: "UM",
			SystemName: "User Mgmt",
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.ClientId, "123")
	c.Set(middlewares.System, "UM")

	GetSystem(sysRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var got model.System
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.SystemCode != "UM" || got.ClientId != "123" {
		t.Fatalf("unexpected system returned: %+v", got)
	}
}

func TestGetSystemNotFoundMapsTo404(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sysRepo := &authTestSystemRepo{err: mongo.ErrNoDocuments}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middlewares.ClientId, "123")
	c.Set(middlewares.System, "UNKNOWN")

	GetSystem(sysRepo)(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestGetSystemsForwardsFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sysRepo := &authTestSystemRepo{
		systemsResult: []model.System{
			{ClientId: "123", SystemCode: "UM"},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/system?clientId=123&systemCode=UM", nil)

	GetSystems(sysRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !sysRepo.getSystemsCalled {
		t.Fatal("expected GetSystems to be called")
	}
	if sysRepo.getSystemsFilter.ClientId != "123" || sysRepo.getSystemsFilter.SystemCode != "UM" {
		t.Fatalf("expected filter forwarded, got %+v", sysRepo.getSystemsFilter)
	}
}

func TestGetSystemsRepositoryErrorMapsTo500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sysRepo := &authTestSystemRepo{systemsErr: errors.New("db down")}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/system", nil)

	GetSystems(sysRepo)(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAddSystemCreates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sysRepo := &authTestSystemRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{
		"clientId":   "123",
		"systemName": "User Mgmt",
		"systemCode": "UM",
		"host":       "https://um.example.com",
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/system", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middlewares.UserId, primitive.NewObjectID().Hex())

	AddSystem(sysRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if sysRepo.createdSystem == nil {
		t.Fatal("expected CreateSystem to be called")
	}
	if sysRepo.createdSystem.SystemCode != "UM" || sysRepo.createdSystem.ClientId != "123" {
		t.Fatalf("unexpected created system: %+v", sysRepo.createdSystem)
	}
}

func TestAddSystemValidatesBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sysRepo := &authTestSystemRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{"clientId": "123"})
	c.Request = httptest.NewRequest(http.MethodPost, "/system", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	AddSystem(sysRepo)(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 validation error, got %d body=%s", w.Code, w.Body.String())
	}
	if sysRepo.createdSystem != nil {
		t.Fatal("expected no creation on validation error")
	}
}

func TestGetSystemByIdReturnsSystem(t *testing.T) {
	gin.SetMode(gin.TestMode)

	targetID := primitive.NewObjectID()
	sysRepo := &authTestSystemRepo{
		getByIdResult: &model.System{Id: targetID, ClientId: "123", SystemCode: "UM"},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: targetID.Hex()}}

	GetSystemById(sysRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if sysRepo.getByIdId != targetID.Hex() {
		t.Fatalf("expected GetSystemById(%s), got %q", targetID.Hex(), sysRepo.getByIdId)
	}
}

func TestGetSystemByIdInvalidHex(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sysRepo := &authTestSystemRepo{getByIdErr: primitive.ErrInvalidHex}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "not-a-hex"}}

	GetSystemById(sysRepo)(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid id, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestGetSystemByIdNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sysRepo := &authTestSystemRepo{getByIdErr: mongo.ErrNoDocuments}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: primitive.NewObjectID().Hex()}}

	GetSystemById(sysRepo)(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteSystemByIdRemoves(t *testing.T) {
	gin.SetMode(gin.TestMode)

	targetID := primitive.NewObjectID()
	sysRepo := &authTestSystemRepo{
		removeSystemRes: &model.System{Id: targetID, ClientId: "123", SystemCode: "UM"},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: targetID.Hex()}}

	DeleteSystemById(sysRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if sysRepo.removedSystemId != targetID.Hex() {
		t.Fatalf("expected remove of %s, got %q", targetID.Hex(), sysRepo.removedSystemId)
	}
}

func TestDeleteSystemByIdNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sysRepo := &authTestSystemRepo{removeErr: mongo.ErrNoDocuments}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: primitive.NewObjectID().Hex()}}

	DeleteSystemById(sysRepo)(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateSystemByIdUpdates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	targetID := primitive.NewObjectID()
	sysRepo := &authTestSystemRepo{
		updateSystemRes: &model.System{Id: targetID, ClientId: "123", SystemCode: "UM", Host: "https://new.example.com"},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{
		"systemName": "User Mgmt v2",
		"host":       "https://new.example.com",
	})
	c.Request = httptest.NewRequest(http.MethodPut, "/system/"+targetID.Hex(), bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: targetID.Hex()}}
	c.Set(middlewares.UserId, primitive.NewObjectID().Hex())

	UpdateSystemById(sysRepo)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if sysRepo.updatedSystemId != targetID.Hex() {
		t.Fatalf("expected update of %s, got %q", targetID.Hex(), sysRepo.updatedSystemId)
	}
}

func TestUpdateSystemByIdValidatesBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sysRepo := &authTestSystemRepo{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(gin.H{"systemName": "only-name"}) // missing host
	c.Request = httptest.NewRequest(http.MethodPut, "/system/x", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: primitive.NewObjectID().Hex()}}

	UpdateSystemById(sysRepo)(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	if sysRepo.updatedSystemId != "" {
		t.Fatal("expected no update on validation error")
	}
}
