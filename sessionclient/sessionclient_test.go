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

func TestCheckReturnsLiveSessionForTheSystem(t *testing.T) {
	store := &fakeStore{session: Session{UserId: "u1", System: "POS"}}
	checker, _ := newChecker(store)

	session, err := checker.Check(context.Background(), "s1", "POS")
	if err != nil || session.UserId != "u1" {
		t.Fatalf("expected live session for u1, got %+v err=%v", session, err)
	}
}

func TestCheckCachesForAtMostTTL(t *testing.T) {
	store := &fakeStore{session: Session{UserId: "u1", System: "POS"}}
	checker, clk := newChecker(store)
	ctx := context.Background()

	_, _ = checker.Check(ctx, "s1", "POS")
	clk.t = clk.t.Add(DefaultTTL - time.Second)
	_, _ = checker.Check(ctx, "s1", "POS")
	if store.calls != 1 {
		t.Fatalf("expected cached answer within TTL, got %d lookups", store.calls)
	}
	clk.t = clk.t.Add(time.Second)
	_, _ = checker.Check(ctx, "s1", "POS")
	if store.calls != 2 {
		t.Fatalf("expected a new lookup after %v, got %d", DefaultTTL, store.calls)
	}
}

func TestRevocationTakesEffectAfterCacheExpires(t *testing.T) {
	store := &fakeStore{session: Session{UserId: "u1", System: "POS"}}
	checker, clk := newChecker(store)
	ctx := context.Background()
	_, _ = checker.Check(ctx, "s1", "POS")

	store.err = ErrSessionRejected
	clk.t = clk.t.Add(DefaultTTL)
	if _, err := checker.Check(ctx, "s1", "POS"); !errors.Is(err, ErrSessionRejected) {
		t.Fatalf("expected ErrSessionRejected, got %v", err)
	}
	store.err = nil
	if _, err := checker.Check(ctx, "s1", "POS"); err != nil || store.calls != 3 {
		t.Fatalf("a rejection must not be cached: err=%v calls=%d", err, store.calls)
	}
}

func TestCheckRejectsSessionForAnotherSystem(t *testing.T) {
	checker, _ := newChecker(&fakeStore{session: Session{UserId: "u1", System: "PHARMACY"}})
	if _, err := checker.Check(context.Background(), "s1", "POS"); !errors.Is(err, ErrSessionRejected) {
		t.Fatalf("expected ErrSessionRejected, got %v", err)
	}
}

func TestCheckAcceptsLegacySessionWithoutSystem(t *testing.T) {
	checker, _ := newChecker(&fakeStore{session: Session{UserId: "u1"}})
	if _, err := checker.Check(context.Background(), "s1", "POS"); err != nil {
		t.Fatalf("expected legacy session to pass, got %v", err)
	}
}

func TestCheckRejectsEmptySessionID(t *testing.T) {
	store := &fakeStore{session: Session{UserId: "u1"}}
	checker, _ := newChecker(store)
	if _, err := checker.Check(context.Background(), "", "POS"); !errors.Is(err, ErrSessionRejected) || store.calls != 0 {
		t.Fatalf("expected rejection without lookup, got %v (%d calls)", err, store.calls)
	}
}

func TestCachedAnswerCoversShortOutageOnly(t *testing.T) {
	store := &fakeStore{session: Session{UserId: "u1", System: "POS"}}
	checker, clk := newChecker(store)
	ctx := context.Background()
	_, _ = checker.Check(ctx, "s1", "POS")

	store.err = ErrUnavailable
	clk.t = clk.t.Add(DefaultTTL / 2)
	if _, err := checker.Check(ctx, "s1", "POS"); err != nil {
		t.Fatalf("expected cached answer during short outage, got %v", err)
	}
	clk.t = clk.t.Add(DefaultTTL)
	session, err := checker.Check(ctx, "s1", "POS")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable once the cache expires, got %v", err)
	}
	if session.UserId != "u1" {
		t.Fatalf("expected the last confirmed session alongside ErrUnavailable, got %+v", session)
	}
}

func TestOutageWithoutPriorConfirmationReturnsNoSession(t *testing.T) {
	checker, _ := newChecker(&fakeStore{err: ErrUnavailable})
	session, err := checker.Check(context.Background(), "s1", "POS")
	if !errors.Is(err, ErrUnavailable) || session != (Session{}) {
		t.Fatalf("expected ErrUnavailable and no session, got %+v err=%v", session, err)
	}
}

func TestStaleSessionIsNotReturnedPastTokenLifetime(t *testing.T) {
	store := &fakeStore{session: Session{UserId: "u1", System: "POS"}}
	checker, clk := newChecker(store)
	ctx := context.Background()
	_, _ = checker.Check(ctx, "s1", "POS")

	store.err = ErrUnavailable
	clk.t = clk.t.Add(staleLimit)
	if session, err := checker.Check(ctx, "s1", "POS"); !errors.Is(err, ErrUnavailable) || session != (Session{}) {
		t.Fatalf("expected no session past %v, got %+v err=%v", staleLimit, session, err)
	}
}

func TestRejectedSessionIsNotReturned(t *testing.T) {
	store := &fakeStore{session: Session{UserId: "u1", System: "POS"}}
	checker, clk := newChecker(store)
	ctx := context.Background()
	_, _ = checker.Check(ctx, "s1", "POS")

	store.err = ErrSessionRejected
	clk.t = clk.t.Add(DefaultTTL)
	if session, err := checker.Check(ctx, "s1", "POS"); !errors.Is(err, ErrSessionRejected) || session != (Session{}) {
		t.Fatalf("expected rejection with no session, got %+v err=%v", session, err)
	}
	store.err = ErrUnavailable
	if session, _ := checker.Check(ctx, "s1", "POS"); session != (Session{}) {
		t.Fatalf("a rejected session must not be served during a later outage, got %+v", session)
	}
}

func TestDisabledCheckerAllowsEverything(t *testing.T) {
	var nilChecker *Checker
	disabled, err := New("")
	if err != nil {
		t.Fatalf("New(\"\"): %v", err)
	}
	for _, c := range []*Checker{nilChecker, disabled, NewChecker(nil)} {
		if _, err := c.Check(context.Background(), "", ""); c.Enabled() || err != nil {
			t.Fatal("expected a disabled checker to allow everything")
		}
	}
}

func TestAuthorizeAppliesTheOutagePolicy(t *testing.T) {
	store := &fakeStore{session: Session{UserId: "u1", System: "POS"}}
	checker, clk := newChecker(store)
	ctx := context.Background()

	if s, err := checker.Authorize(ctx, "s1", "POS", http.MethodPost); err != nil || s.UserId != "u1" {
		t.Fatalf("live session: got %+v err=%v", s, err)
	}

	store.err = ErrUnavailable
	clk.t = clk.t.Add(DefaultTTL)
	if s, err := checker.Authorize(ctx, "s1", "POS", http.MethodGet); err != nil || s.UserId != "u1" {
		t.Fatalf("outage GET with a known session should continue, got %+v err=%v", s, err)
	}
	if _, err := checker.Authorize(ctx, "s1", "POS", http.MethodPost); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("outage POST must be refused, got %v", err)
	}
	if _, err := checker.Authorize(ctx, "never-seen", "POS", http.MethodGet); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("outage GET for an unknown session must be refused, got %v", err)
	}

	store.err = ErrSessionRejected
	if _, err := checker.Authorize(ctx, "s1", "POS", http.MethodGet); !errors.Is(err, ErrSessionRejected) {
		t.Fatalf("a revoked session must be refused, got %v", err)
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
