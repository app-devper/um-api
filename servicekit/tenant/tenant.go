// Package tenant keeps one database per tenant (UM client id), named
// <prefix>_<clientId>, with client 000 using <prefix> itself (um-api
// ADR-0007). It is independent of the database driver: a service supplies how
// to open a database by name and how to initialise it (indexes, seed data).
//
// Initialisation runs once per tenant per process. If it fails the tenant is
// still served, and it is tried again on a later request, at most once per
// RetryAfter, so a stubborn failure neither blocks the tenant nor slows every
// request.
package tenant

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"sync"
	"time"
)

var validClientID = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9_-]{0,48}[a-zA-Z0-9])?$`)

// SuperClientID is the client whose database is the bare prefix.
const SuperClientID = "000"

// RetryAfter is the least time between initialisation attempts of a tenant.
const RetryAfter = time.Minute

// InitTimeout bounds one initialisation attempt.
const InitTimeout = 30 * time.Second

// ValidateClientID refuses an empty or malformed client id.
func ValidateClientID(clientID string) error {
	if clientID == "" {
		return errors.New("clientId is required")
	}
	if !validClientID.MatchString(clientID) {
		return fmt.Errorf("invalid clientId %q", clientID)
	}
	return nil
}

// DatabaseName is the tenant's database: prefix for client 000, else
// prefix_clientId.
func DatabaseName(prefix, clientID string) string {
	if clientID == SuperClientID {
		return prefix
	}
	return prefix + "_" + clientID
}

type entry[DB any] struct {
	db          DB
	mu          sync.Mutex
	initialised bool
	lastTry     time.Time
}

// Registry opens and caches each tenant's database.
type Registry[DB any] struct {
	prefix  string
	open    func(name string) DB
	init    func(ctx context.Context, clientID string, db DB) error
	now     func() time.Time
	entries sync.Map
}

// New makes a registry. open returns a handle for a database name; init, if
// not nil, prepares a tenant's database and may be called again after an
// error.
func New[DB any](prefix string, open func(name string) DB, init func(ctx context.Context, clientID string, db DB) error) *Registry[DB] {
	return &Registry[DB]{prefix: prefix, open: open, init: init, now: time.Now}
}

// For returns the tenant's database, initialising it on first use.
func (r *Registry[DB]) For(clientID string) (DB, error) {
	var zero DB
	if err := ValidateClientID(clientID); err != nil {
		return zero, err
	}
	v, ok := r.entries.Load(clientID)
	if !ok {
		v, _ = r.entries.LoadOrStore(clientID, &entry[DB]{db: r.open(DatabaseName(r.prefix, clientID))})
	}
	e := v.(*entry[DB])
	r.ensureInitialised(clientID, e)
	return e.db, nil
}

// Known lists the tenants this process has opened, sorted.
func (r *Registry[DB]) Known() []string {
	ids := []string{}
	r.entries.Range(func(k, _ any) bool {
		ids = append(ids, k.(string))
		return true
	})
	sort.Strings(ids)
	return ids
}

func (r *Registry[DB]) ensureInitialised(clientID string, e *entry[DB]) {
	if r.init == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.initialised || (!e.lastTry.IsZero() && r.now().Sub(e.lastTry) < RetryAfter) {
		return
	}
	e.lastTry = r.now()
	ctx, cancel := context.WithTimeout(context.Background(), InitTimeout)
	defer cancel()
	if err := r.init(ctx, clientID, e.db); err != nil {
		slog.Warn("tenant initialisation failed; will retry", "client", clientID, "error", err)
		return
	}
	e.initialised = true
}
