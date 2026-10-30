package session

import (
	"errors"
	"testing"
	"um/app/core/constant"
	"um/app/domain/model"
)

func launchLogin(t *testing.T) *login {
	t.Helper()
	l := newLogin(t)
	sys := l.m.systems.(*systems)
	sys.byClient["123"] = []string{"SM", "POS", "PHARMACY", "GOLD", "EVIL", "UNLISTED"}
	l.m.catalog = &catalog{entries: []model.SystemCatalogEntry{
		{SystemCode: "SM", SystemName: "User Management", WebUrl: "https://um.devper.app"},
		{SystemCode: "POS", SystemName: "Devper POS", WebUrl: "https://pos.devper.app/some/path?x=1"},
		{SystemCode: "PHARMACY", SystemName: "Pharmacy App"},
		{SystemCode: "GOLD", SystemName: "Devper Gold", WebUrl: "http://localhost:5173"},
		{SystemCode: "EVIL", SystemName: "Evil", WebUrl: "javascript:alert(1)"},
	}}
	return l
}

func TestLaunchableListsOtherSystemsWithTheCatalogsNameAndWebUrl(t *testing.T) {
	l := launchLogin(t)
	u := l.user(t, "alice", constant.USER, "123", constant.ACTIVE)

	got, err := l.m.Launchable(u.Id.Hex(), "SM")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]model.System{
		"POS":  {SystemCode: "POS", SystemName: "Devper POS", WebUrl: "https://pos.devper.app"},
		"GOLD": {SystemCode: "GOLD", SystemName: "Devper Gold", WebUrl: "http://localhost:5173"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %v", got, want)
	}
	for _, s := range got {
		w := want[s.SystemCode]
		if s.WebUrl != w.WebUrl || s.SystemName != w.SystemName {
			t.Fatalf("%s: got %q %q, want %q %q", s.SystemCode, s.SystemName, s.WebUrl, w.SystemName, w.WebUrl)
		}
	}
}

func TestLaunchableRefusesAnInactiveUser(t *testing.T) {
	l := launchLogin(t)
	u := l.user(t, "bob", constant.USER, "123", constant.INACTIVE)

	if _, err := l.m.Launchable(u.Id.Hex(), "SM"); !errors.Is(err, ErrUserInactive) {
		t.Fatalf("got %v, want ErrUserInactive", err)
	}
}

func TestLaunchableGivesASuperNothingOutsideSM(t *testing.T) {
	l := launchLogin(t)
	u := l.user(t, "root", constant.SUPER, "123", constant.ACTIVE)

	got, err := l.m.Launchable(u.Id.Hex(), "SM")
	if err != nil || len(got) != 0 {
		t.Fatalf("got %+v, %v; want nothing", got, err)
	}
}

func TestLaunchableWithoutACatalogListsNothing(t *testing.T) {
	l := launchLogin(t)
	l.m.catalog = nil
	u := l.user(t, "alice", constant.USER, "123", constant.ACTIVE)

	got, err := l.m.Launchable(u.Id.Hex(), "SM")
	if err != nil || len(got) != 0 {
		t.Fatalf("got %+v, %v; want nothing", got, err)
	}
}

func TestAdmitAppliesTheLoginRules(t *testing.T) {
	l := launchLogin(t)
	active := l.user(t, "alice", constant.USER, "123", constant.ACTIVE)
	inactive := l.user(t, "bob", constant.USER, "123", constant.INACTIVE)
	super := l.user(t, "root", constant.SUPER, "123", constant.ACTIVE)

	cases := []struct {
		name   string
		userId string
		system string
		want   error
	}{
		{"a System of the Client", active.Id.Hex(), "POS", nil},
		{"a System the Client lacks", active.Id.Hex(), "SNOOK", ErrWrongSystem},
		{"an inactive User", inactive.Id.Hex(), "POS", ErrNotActive},
		{"a SUPER outside SM", super.Id.Hex(), "POS", ErrWrongSystem},
		{"an unknown User", "000000000000000000000000", "POS", ErrSessionInvalid},
	}
	for _, c := range cases {
		if err := l.m.Admit(c.userId, c.system); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
}

func TestResumeRefusesASystemTheClientLacks(t *testing.T) {
	l := launchLogin(t)
	u := l.user(t, "alice", constant.USER, "123", constant.ACTIVE)

	if _, err := l.m.Resume(u.Id.Hex(), "SNOOK", Metadata{}); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("got %v, want ErrSessionInvalid", err)
	}
	if _, err := l.m.Resume(u.Id.Hex(), "POS", Metadata{}); err != nil {
		t.Fatalf("resume on a System of the Client: %v", err)
	}
}
