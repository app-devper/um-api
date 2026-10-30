package systemcatalog

import (
	"testing"
	"um/app/domain/model"
)

func TestSeedTakesOneEntryPerCodePreferringTheSuperClient(t *testing.T) {
	systems := []model.System{
		{ClientId: "001", SystemCode: "POS", SystemName: "Devper POS 001", Host: "https://pos-001.example"},
		{ClientId: "000", SystemCode: "POS", SystemName: "Devper POS", Host: "https://pos.example", WebUrl: "https://pos.devper.app"},
		{ClientId: "002", SystemCode: "PHARMACY", SystemName: "Pharmacy App", Host: "https://api.example"},
		{ClientId: "004", SystemCode: "PHARMACY", SystemName: "Pharmacy App 4", Host: "https://other.example"},
	}

	got := Seed(systems)

	want := map[string]model.SystemCatalogEntry{
		"POS":      {SystemCode: "POS", SystemName: "Devper POS", Host: "https://pos.example", WebUrl: "https://pos.devper.app"},
		"PHARMACY": {SystemCode: "PHARMACY", SystemName: "Pharmacy App", Host: "https://api.example"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d: %+v", len(got), len(want), got)
	}
	for _, e := range got {
		if want[e.SystemCode] != e {
			t.Errorf("%s: got %+v, want %+v", e.SystemCode, e, want[e.SystemCode])
		}
	}
}

func TestSeedOfNoSystemsIsEmpty(t *testing.T) {
	if got := Seed(nil); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}
