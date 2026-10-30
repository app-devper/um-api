// Package systemcatalog owns the System catalog (ADR-0012): one entry per
// System code that every Client's System of that code follows.
package systemcatalog

import (
	"sort"
	"um/app/core/constant"
	"um/app/domain/model"
)

// Seed builds a catalog entry for each System code found among the Clients'
// Systems, taking the name, API host and web URL from the super Client's row
// when it has one, otherwise from the first row of that code.
func Seed(systems []model.System) []model.SystemCatalogEntry {
	chosen := map[string]model.System{}
	for _, s := range systems {
		current, seen := chosen[s.SystemCode]
		if !seen || (current.ClientId != constant.SuperClientId && s.ClientId == constant.SuperClientId) {
			chosen[s.SystemCode] = s
		}
	}
	entries := make([]model.SystemCatalogEntry, 0, len(chosen))
	for code, s := range chosen {
		entries = append(entries, model.SystemCatalogEntry{SystemCode: code, SystemName: s.SystemName, Host: s.Host, WebUrl: s.WebUrl})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].SystemCode < entries[j].SystemCode })
	return entries
}
