package config

import "testing"

func TestLoadAppConfigReturnsValidatedConfig(t *testing.T) {
	t.Setenv("PORT", "8585")
	t.Setenv("MONGO_HOST", "mongodb://localhost:27017")
	t.Setenv("MONGO_UM_DB_NAME", "um")
	t.Setenv("REDIS_HOST", "localhost:6379")
	t.Setenv("SECRET_KEY", "secret")
	t.Setenv("LOGIN_LOCKOUT_ENABLED", "true")

	cfg, err := LoadAppConfig()
	if err != nil {
		t.Fatalf("LoadAppConfig returned error: %v", err)
	}
	if cfg.Port != "8585" {
		t.Fatalf("expected port 8585, got %q", cfg.Port)
	}
	if !cfg.LockoutEnabled {
		t.Fatal("expected lockout to be enabled")
	}
	if cfg.ListenAddr() != ":8585" {
		t.Fatalf("expected listen addr :8585, got %q", cfg.ListenAddr())
	}
}

func TestLoadAppConfigRejectsMissingRequiredEnv(t *testing.T) {
	t.Setenv("PORT", "8585")
	t.Setenv("MONGO_HOST", "")
	t.Setenv("MONGO_UM_DB_NAME", "um")
	t.Setenv("REDIS_HOST", "localhost:6379")
	t.Setenv("SECRET_KEY", "secret")

	_, err := LoadAppConfig()
	if err == nil {
		t.Fatal("expected missing env to fail validation")
	}
	if err.Error() != "MONGO_HOST is required" {
		t.Fatalf("expected MONGO_HOST validation error, got %q", err.Error())
	}
}

func TestLoadAppConfigRejectsInvalidPort(t *testing.T) {
	t.Setenv("PORT", "70000")
	t.Setenv("MONGO_HOST", "mongodb://localhost:27017")
	t.Setenv("MONGO_UM_DB_NAME", "um")
	t.Setenv("REDIS_HOST", "localhost:6379")
	t.Setenv("SECRET_KEY", "secret")

	_, err := LoadAppConfig()
	if err == nil {
		t.Fatal("expected invalid port to fail validation")
	}
	if err.Error() != "PORT must be a valid TCP port" {
		t.Fatalf("expected port validation error, got %q", err.Error())
	}
}

func TestLoadAppConfigAcceptsMongoAndRedisURLs(t *testing.T) {
	t.Setenv("PORT", "8585")
	t.Setenv("MONGO_HOST", "mongodb://localhost:27017")
	t.Setenv("MONGO_UM_DB_NAME", "um")
	t.Setenv("REDIS_HOST", "redis://localhost:6379/0")
	t.Setenv("SECRET_KEY", "secret")

	if _, err := LoadAppConfig(); err != nil {
		t.Fatalf("expected URLs to pass validation, got %v", err)
	}
}

func TestLoadAppConfigAcceptsHostPortValues(t *testing.T) {
	t.Setenv("PORT", "8585")
	t.Setenv("MONGO_HOST", "localhost:27017")
	t.Setenv("MONGO_UM_DB_NAME", "um")
	t.Setenv("REDIS_HOST", "localhost:6379")
	t.Setenv("SECRET_KEY", "secret")

	if _, err := LoadAppConfig(); err != nil {
		t.Fatalf("expected host:port values to pass validation, got %v", err)
	}
}

func TestLoadAppConfigRejectsInvalidMongoHost(t *testing.T) {
	t.Setenv("PORT", "8585")
	t.Setenv("MONGO_HOST", "not-a-valid-mongo-uri")
	t.Setenv("MONGO_UM_DB_NAME", "um")
	t.Setenv("REDIS_HOST", "localhost:6379")
	t.Setenv("SECRET_KEY", "secret")

	_, err := LoadAppConfig()
	if err == nil {
		t.Fatal("expected invalid mongo host to fail validation")
	}
	if err.Error() != "MONGO_HOST must be a valid MongoDB URI or host:port" {
		t.Fatalf("expected mongo host validation error, got %q", err.Error())
	}
}

func TestLoadAppConfigRejectsInvalidRedisHost(t *testing.T) {
	t.Setenv("PORT", "8585")
	t.Setenv("MONGO_HOST", "localhost:27017")
	t.Setenv("MONGO_UM_DB_NAME", "um")
	t.Setenv("REDIS_HOST", "redis://")
	t.Setenv("SECRET_KEY", "secret")

	_, err := LoadAppConfig()
	if err == nil {
		t.Fatal("expected invalid redis host to fail validation")
	}
	if err.Error() != "REDIS_HOST must be a valid Redis URL or host:port" {
		t.Fatalf("expected redis host validation error, got %q", err.Error())
	}
}

func TestWebAuthnOriginsMustBeHTTPSOnTheRPID(t *testing.T) {
	base := AppConfig{Port: "8585", MongoHost: "localhost:27017", MongoUMDBName: "um", RedisHost: "localhost:6379", SecretKey: "s"}
	cases := []struct {
		name    string
		rpID    string
		origins []string
		ok      bool
	}{
		{"disabled", "", nil, true},
		{"subdomain origins", "devper.app", []string{"https://port.devper.app", "https://um.devper.app"}, true},
		{"rp id itself", "devper.app", []string{"https://devper.app"}, true},
		{"localhost dev", "localhost", []string{"http://localhost:4173"}, true},
		{"no origins", "devper.app", nil, false},
		{"plain http", "devper.app", []string{"http://port.devper.app"}, false},
		{"other domain", "devper.app", []string{"https://devper-port.web.app"}, false},
		{"lookalike suffix", "devper.app", []string{"https://evildevper.app"}, false},
		{"origin with path", "devper.app", []string{"https://port.devper.app/login"}, false},
	}
	for _, c := range cases {
		cfg := base
		cfg.WebAuthnRPID, cfg.WebAuthnOrigin = c.rpID, c.origins
		if err := cfg.Validate(); (err == nil) != c.ok {
			t.Errorf("%s: err %v, want ok=%v", c.name, err, c.ok)
		}
	}
}

func TestWebAuthnOriginsAreACommaList(t *testing.T) {
	if got := splitList(" https://a.devper.app, ,https://b.devper.app "); len(got) != 2 || got[1] != "https://b.devper.app" {
		t.Fatalf("got %q", got)
	}
}
