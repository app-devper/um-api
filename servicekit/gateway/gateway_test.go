package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func request(forwarded string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	if forwarded != "" {
		r.Header.Set("X-Forwarded-Host", forwarded)
	}
	return r
}

func TestHostsAllowOnlyTheGateway(t *testing.T) {
	h := ParseHosts(" api.devper.app , Devper-API.web.app,, ")
	cases := map[string]bool{
		"api.devper.app":                 true,
		"API.DEVPER.APP":                 true,
		"devper-api.web.app":             true,
		"api.devper.app, proxy.internal": true,
		"pharmacy-xyz.a.run.app":         false,
		"":                               false,
	}
	for forwarded, want := range cases {
		if got := h.Allows(request(forwarded)); got != want {
			t.Errorf("%q: allowed=%v, want %v", forwarded, got, want)
		}
	}
}

func TestNoHostsAllowsEverything(t *testing.T) {
	h := ParseHosts("")
	if h.Enabled() || !h.Allows(request("")) {
		t.Fatal("an empty list must allow every request")
	}
}

func TestMiddlewareUsesTheServicesRefusal(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	refuse := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errcode":"GW-403","error":"` + Message + `"}`))
	}
	handler := Middleware(ParseHosts("api.devper.app"), refuse)(next)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, request("run.app"))
	if rec.Code != http.StatusForbidden || rec.Body.String() != `{"errcode":"GW-403","error":"direct access is not allowed"}` {
		t.Fatalf("refused: %d %s", rec.Code, rec.Body)
	}
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, request("api.devper.app"))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("allowed: %d", rec.Code)
	}
}
