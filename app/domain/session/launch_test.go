package session

import (
	"errors"
	"testing"
	"um/app/core/constant"
)

func launchLogin(t *testing.T) *login {
	t.Helper()
	l := newLogin(t)
	sys := l.m.systems.(*systems)
	sys.byClient["123"] = []string{"SM", "POS", "PHARMACY", "GOLD", "EVIL"}
	sys.hosts = map[string]string{
		"SM":       "https://um.devper.app",
		"POS":      "https://pos.devper.app/some/path?x=1",
		"PHARMACY": "",
		"GOLD":     "http://localhost:5173",
		"EVIL":     "javascript:alert(1)",
	}
	return l
}

func TestLaunchableListsOtherSystemsWithAUsableWebUrl(t *testing.T) {
	l := launchLogin(t)
	u := l.user(t, "alice", constant.USER, "123", constant.ACTIVE)

	got, err := l.m.Launchable(u.Id.Hex(), "SM")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"POS": "https://pos.devper.app", "GOLD": "http://localhost:5173"}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %v", got, want)
	}
	for _, s := range got {
		if want[s.SystemCode] != s.WebUrl {
			t.Fatalf("%s: webUrl %q, want %q", s.SystemCode, s.WebUrl, want[s.SystemCode])
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
