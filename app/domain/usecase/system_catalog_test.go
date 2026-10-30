package usecase

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"um/app/core/errs"
	"um/app/domain/model"
	"um/app/domain/repository"
	"um/middlewares"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type fakeCatalog struct {
	repository.ISystemCatalog
	entries map[string]model.SystemCatalogEntry
	saved   *model.SystemCatalogEntry
	deleted string
}

func newFakeCatalog(entries ...model.SystemCatalogEntry) *fakeCatalog {
	c := &fakeCatalog{entries: map[string]model.SystemCatalogEntry{}}
	for _, e := range entries {
		c.entries[e.SystemCode] = e
	}
	return c
}

func (c *fakeCatalog) List() ([]model.SystemCatalogEntry, error) {
	items := []model.SystemCatalogEntry{}
	for _, e := range c.entries {
		items = append(items, e)
	}
	return items, nil
}

func (c *fakeCatalog) Get(code string) (*model.SystemCatalogEntry, error) {
	e, ok := c.entries[code]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}
	return &e, nil
}

func (c *fakeCatalog) Save(e model.SystemCatalogEntry) (*model.SystemCatalogEntry, error) {
	c.saved = &e
	c.entries[e.SystemCode] = e
	return &e, nil
}

func (c *fakeCatalog) Delete(code string) error {
	if _, ok := c.entries[code]; !ok {
		return mongo.ErrNoDocuments
	}
	c.deleted = code
	delete(c.entries, code)
	return nil
}

var pharmacyEntry = model.SystemCatalogEntry{SystemCode: "PHARMACY", SystemName: "Pharmacy App", WebUrl: "https://pharm.devper.app", Host: "https://api.devper.app"}

func jsonRequest(method, path string, body any) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	raw, _ := json.Marshal(body)
	c.Request = httptest.NewRequest(method, path, bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middlewares.UserId, primitive.NewObjectID().Hex())
	return c, w
}

func TestSaveSystemCatalogEntryWritesTheCodeFromThePath(t *testing.T) {
	catalog := newFakeCatalog()
	c, w := jsonRequest(http.MethodPut, "/system-catalog/GOLD", gin.H{"systemName": "Devper Gold", "webUrl": "https://gold.devper.app", "host": "https://gold-api.example"})
	c.Params = gin.Params{{Key: "code", Value: "GOLD"}}

	SaveSystemCatalogEntry(catalog)(c)

	if w.Code != http.StatusOK || catalog.saved == nil {
		t.Fatalf("got %d body=%s", w.Code, w.Body.String())
	}
	want := model.SystemCatalogEntry{SystemCode: "GOLD", SystemName: "Devper Gold", WebUrl: "https://gold.devper.app", Host: "https://gold-api.example"}
	if *catalog.saved != want {
		t.Fatalf("saved %+v, want %+v", *catalog.saved, want)
	}
}

func TestSaveSystemCatalogEntryRefusesABadCodeOrUrl(t *testing.T) {
	cases := []struct {
		code string
		body gin.H
	}{
		{"gold", gin.H{"systemName": "x"}},
		{"GOLD", gin.H{"systemName": "x", "webUrl": "not a url"}},
		{"GOLD", gin.H{"webUrl": "https://gold.devper.app"}},
	}
	for _, tc := range cases {
		catalog := newFakeCatalog()
		c, w := jsonRequest(http.MethodPut, "/system-catalog/"+tc.code, tc.body)
		c.Params = gin.Params{{Key: "code", Value: tc.code}}

		SaveSystemCatalogEntry(catalog)(c)

		if w.Code != http.StatusBadRequest || catalog.saved != nil {
			t.Errorf("%s %v: got %d body=%s", tc.code, tc.body, w.Code, w.Body.String())
		}
	}
}

func TestDeleteSystemCatalogEntryRefusesACodeClientsStillUse(t *testing.T) {
	catalog := newFakeCatalog(pharmacyEntry)
	systems := &authTestSystemRepo{byCode: map[string][]model.System{"PHARMACY": {{ClientId: "002", SystemCode: "PHARMACY"}}}}
	c, w := jsonRequest(http.MethodDelete, "/system-catalog/PHARMACY", nil)
	c.Params = gin.Params{{Key: "code", Value: "PHARMACY"}}

	DeleteSystemCatalogEntry(catalog, systems)(c)

	if w.Code != http.StatusConflict || !bytes.Contains(w.Body.Bytes(), []byte(errs.ErrSystemInUse)) || catalog.deleted != "" {
		t.Fatalf("got %d body=%s deleted=%q", w.Code, w.Body.String(), catalog.deleted)
	}
}

func TestDeleteSystemCatalogEntryRemovesAnUnusedCode(t *testing.T) {
	catalog := newFakeCatalog(pharmacyEntry)
	c, w := jsonRequest(http.MethodDelete, "/system-catalog/PHARMACY", nil)
	c.Params = gin.Params{{Key: "code", Value: "PHARMACY"}}

	DeleteSystemCatalogEntry(catalog, &authTestSystemRepo{})(c)

	if w.Code != http.StatusOK || catalog.deleted != "PHARMACY" {
		t.Fatalf("got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAddSystemTakesTheNameAndDefaultHostFromTheCatalog(t *testing.T) {
	systems := &authTestSystemRepo{}
	c, w := jsonRequest(http.MethodPost, "/system", gin.H{"clientId": "004", "systemCode": "PHARMACY"})

	AddSystem(systems, newFakeCatalog(pharmacyEntry))(c)

	if w.Code != http.StatusOK || systems.createdSystem == nil {
		t.Fatalf("got %d body=%s", w.Code, w.Body.String())
	}
	got := systems.createdSystem
	if got.ClientId != "004" || got.SystemName != "Pharmacy App" || got.Host != "https://api.devper.app" {
		t.Fatalf("created %+v", got)
	}
}

func TestAddSystemKeepsAHostGivenForTheClient(t *testing.T) {
	systems := &authTestSystemRepo{}
	c, w := jsonRequest(http.MethodPost, "/system", gin.H{"clientId": "004", "systemCode": "PHARMACY", "host": "https://pharm-004.example"})

	AddSystem(systems, newFakeCatalog(pharmacyEntry))(c)

	if w.Code != http.StatusOK || systems.createdSystem.Host != "https://pharm-004.example" {
		t.Fatalf("got %d body=%s created=%+v", w.Code, w.Body.String(), systems.createdSystem)
	}
}

func TestAddSystemRefusesACodeOutsideTheCatalog(t *testing.T) {
	systems := &authTestSystemRepo{}
	c, w := jsonRequest(http.MethodPost, "/system", gin.H{"clientId": "004", "systemCode": "NOPE"})

	AddSystem(systems, newFakeCatalog(pharmacyEntry))(c)

	if w.Code != http.StatusBadRequest || systems.createdSystem != nil {
		t.Fatalf("got %d body=%s", w.Code, w.Body.String())
	}
}
