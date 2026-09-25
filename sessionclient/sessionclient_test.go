package sessionclient

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

type fakeStore struct {
	session Session
	err     error
	calls   int
}

func (f *fakeStore) Session(context.Context, string) (Session, error) {
	f.calls++
	return f.session, f.err
}

type clock struct{ t time.Time }

func newChecker(store Store) (*Checker, *clock) {
	c := &clock{t: time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)}
	checker := NewChecker(store)
	checker.now = func() time.Time { return c.t }
	return checker, c
}

func TestCheckAcceptsLiveSessionForTheSystem(t *testing.T) {
	store := &fakeStore{session: Session{UserId: "u1", System: "POS"}}
	checker, _ := newChecker(store)

	if err := checker.Check(context.Background(), "s1", "POS"); err != nil {
		t.Fatalf("expected live session, got %v", err)
	}
}

func TestCheckCachesForAtMostTTL(t *testing.T) {
	store := &fakeStore{session: Session{UserId: "u1", System: "POS"}}
	checker, clk := newChecker(store)
	ctx := context.Background()

	_ = checker.Check(ctx, "s1", "POS")
	clk.t = clk.t.Add(DefaultTTL - time.Second)
	_ = checker.Check(ctx, "s1", "POS")
	if store.calls != 1 {
		t.Fatalf("expected cached answer within TTL, got %d lookups", store.calls)
	}
	clk.t = clk.t.Add(time.Second)
	_ = checker.Check(ctx, "s1", "POS")
	if store.calls != 2 {
		t.Fatalf("expected a new lookup after %v, got %d", DefaultTTL, store.calls)
	}
}

func TestRevocationTakesEffectAfterCacheExpires(t *testing.T) {
	store := &fakeStore{session: Session{UserId: "u1", System: "POS"}}
	checker, clk := newChecker(store)
	ctx := context.Background()
	_ = checker.Check(ctx, "s1", "POS")

	store.err = ErrSessionRejected
	clk.t = clk.t.Add(DefaultTTL)
	if err := checker.Check(ctx, "s1", "POS"); !errors.Is(err, ErrSessionRejected) {
		t.Fatalf("expected ErrSessionRejected, got %v", err)
	}
	store.err = nil
	if err := checker.Check(ctx, "s1", "POS"); err != nil || store.calls != 3 {
		t.Fatalf("a rejection must not be cached: err=%v calls=%d", err, store.calls)
	}
}

func TestCheckRejectsSessionForAnotherSystem(t *testing.T) {
	checker, _ := newChecker(&fakeStore{session: Session{UserId: "u1", System: "PHARMACY"}})
	if err := checker.Check(context.Background(), "s1", "POS"); !errors.Is(err, ErrSessionRejected) {
		t.Fatalf("expected ErrSessionRejected, got %v", err)
	}
}

func TestCheckAcceptsLegacySessionWithoutSystem(t *testing.T) {
	checker, _ := newChecker(&fakeStore{session: Session{UserId: "u1"}})
	if err := checker.Check(context.Background(), "s1", "POS"); err != nil {
		t.Fatalf("expected legacy session to pass, got %v", err)
	}
}

func TestCheckRejectsEmptySessionID(t *testing.T) {
	store := &fakeStore{session: Session{UserId: "u1"}}
	checker, _ := newChecker(store)
	if err := checker.Check(context.Background(), "", "POS"); !errors.Is(err, ErrSessionRejected) || store.calls != 0 {
		t.Fatalf("expected rejection without lookup, got %v (%d calls)", err, store.calls)
	}
}

func TestCachedAnswerCoversShortOutageOnly(t *testing.T) {
	store := &fakeStore{session: Session{UserId: "u1", System: "POS"}}
	checker, clk := newChecker(store)
	ctx := context.Background()
	_ = checker.Check(ctx, "s1", "POS")

	store.err = ErrUnavailable
	clk.t = clk.t.Add(DefaultTTL / 2)
	if err := checker.Check(ctx, "s1", "POS"); err != nil {
		t.Fatalf("expected cached answer during short outage, got %v", err)
	}
	clk.t = clk.t.Add(DefaultTTL)
	if err := checker.Check(ctx, "s1", "POS"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable once the cache expires, got %v", err)
	}
}

func TestDisabledCheckerAllowsEverything(t *testing.T) {
	var nilChecker *Checker
	disabled, err := New("")
	if err != nil {
		t.Fatalf("New(\"\"): %v", err)
	}
	for _, c := range []*Checker{nilChecker, disabled, NewChecker(nil)} {
		if c.Enabled() || c.Check(context.Background(), "", "") != nil {
			t.Fatal("expected a disabled checker to allow everything")
		}
	}
}

func TestReadOnlyDuringOutage(t *testing.T) {
	for method, want := range map[string]bool{
		http.MethodGet: true, http.MethodHead: true, http.MethodOptions: true,
		http.MethodPost: false, http.MethodPut: false, http.MethodPatch: false, http.MethodDelete: false,
	} {
		if got := ReadOnlyDuringOutage(method); got != want {
			t.Errorf("%s: got %v, want %v", method, got, want)
		}
	}
}

func TestRedisStoreReadsUMSessionKeys(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb, err := NewRedisClient(mr.Addr())
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	defer rdb.Close()
	store := NewRedisStore(rdb)
	ctx := context.Background()

	mr.Set(KeyPrefix+"s1", `{"userId":"u1","createdAt":"2026-09-25T10:00:00Z","system":"POS"}`)
	if s, err := store.Session(ctx, "s1"); err != nil || s != (Session{UserId: "u1", System: "POS"}) {
		t.Fatalf("unexpected session %+v err=%v", s, err)
	}
	if _, err := store.Session(ctx, "missing"); !errors.Is(err, ErrSessionRejected) {
		t.Fatalf("missing session: expected ErrSessionRejected, got %v", err)
	}
	mr.Set(KeyPrefix+"bad", "not json")
	if _, err := store.Session(ctx, "bad"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unreadable session: expected ErrUnavailable, got %v", err)
	}
	mr.SetError("LOADING")
	if _, err := store.Session(ctx, "s1"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("redis error: expected ErrUnavailable, got %v", err)
	}
}

func TestRedisStoreFailsFastWhenRedisStalls(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			defer conn.Close()
		}
	}()
	rdb, _ := NewRedisClient(ln.Addr().String())
	defer rdb.Close()

	start := time.Now()
	_, err = NewRedisStore(rdb).Session(context.Background(), "s1")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > lookupTimeout+500*time.Millisecond {
		t.Fatalf("lookup took %v, want about %v", elapsed, lookupTimeout)
	}
}

func TestNewRedisClientAcceptsURL(t *testing.T) {
	rdb, err := NewRedisClient("redis://:secret@10.0.0.5:6380/2")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	defer rdb.Close()
	opts := rdb.Options()
	if opts.Addr != "10.0.0.5:6380" || opts.Password != "secret" || opts.DB != 2 {
		t.Fatalf("unexpected options addr=%s db=%d", opts.Addr, opts.DB)
	}
	if !opts.ContextTimeoutEnabled || opts.ReadTimeout != lookupTimeout || opts.MaxRetries > 0 {
		t.Fatalf("lookup must be bounded: ctxTimeout=%v read=%v retries=%d", opts.ContextTimeoutEnabled, opts.ReadTimeout, opts.MaxRetries)
	}
}
