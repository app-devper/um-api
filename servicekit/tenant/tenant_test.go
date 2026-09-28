package tenant

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeDB struct{ name string }

func TestDatabaseNamesAndValidation(t *testing.T) {
	if DatabaseName("gold_shop", "000") != "gold_shop" || DatabaseName("gold_shop", "005") != "gold_shop_005" {
		t.Fatal("database names")
	}
	for _, bad := range []string{"", "a/b", "_x", "x_", "a b", "$where"} {
		if ValidateClientID(bad) == nil {
			t.Errorf("%q should be refused", bad)
		}
	}
	for _, ok := range []string{"000", "005", "PHA", "a-b_c"} {
		if err := ValidateClientID(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
}

func TestEachTenantIsOpenedAndInitialisedOnce(t *testing.T) {
	var opened, inits atomic.Int32
	r := New("alert", func(name string) *fakeDB { opened.Add(1); return &fakeDB{name} },
		func(context.Context, string, *fakeDB) error { inits.Add(1); return nil })

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			db, err := r.For("005")
			if err != nil || db.name != "alert_005" {
				t.Errorf("db %v err %v", db, err)
			}
		}()
	}
	wg.Wait()
	if inits.Load() != 1 {
		t.Fatalf("initialised %d times", inits.Load())
	}
	if _, err := r.For("bad id"); err == nil {
		t.Fatal("bad id accepted")
	}
	if got := r.Known(); len(got) != 1 || got[0] != "005" {
		t.Fatalf("known %v", got)
	}
}

// A failed initialisation still serves the tenant and is retried, but not
// on every request.
func TestFailedInitialisationIsRetriedAfterAWhile(t *testing.T) {
	now := time.Unix(0, 0)
	fail := true
	var attempts int
	r := New("pharmacy", func(name string) *fakeDB { return &fakeDB{name} },
		func(context.Context, string, *fakeDB) error {
			attempts++
			if fail {
				return errors.New("index conflict")
			}
			return nil
		})
	r.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if _, err := r.For("005"); err != nil {
			t.Fatal(err)
		}
	}
	if attempts != 1 {
		t.Fatalf("%d attempts before RetryAfter, want 1", attempts)
	}
	now = now.Add(RetryAfter)
	fail = false
	_, _ = r.For("005")
	_, _ = r.For("005")
	now = now.Add(2 * RetryAfter)
	_, _ = r.For("005")
	if attempts != 2 {
		t.Fatalf("%d attempts, want 2 (one retry, then initialised)", attempts)
	}
}
