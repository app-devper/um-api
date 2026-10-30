package session

import (
	"errors"
	"net/url"
	"um/app/core/constant"
	"um/app/domain/model"
	"um/app/domain/repository"
)

// WithCatalog gives the Manager the System catalog, which says where each
// System's front end is served (ADR-0012).
func WithCatalog(catalog repository.ISystemCatalog) Option {
	return func(m *Manager) { m.catalog = catalog }
}

// Launchable lists the Systems a User may open from the current one (ADR-0011):
// the other Systems of the User's Client that the User may use and whose
// catalog entry has a usable WebUrl (ADR-0012). Name and WebUrl come from the
// catalog; each WebUrl is reduced to its origin.
func (m *Manager) Launchable(userId, current string) ([]model.System, error) {
	if m.systems == nil {
		return nil, errors.New("session manager has no login (WithLogin)")
	}
	user, err := m.users.GetUserById(userId)
	if err != nil || user == nil {
		return nil, ErrSessionInvalid
	}
	if user.Status != constant.ACTIVE {
		return nil, ErrUserInactive
	}
	launchable := []model.System{}
	if m.catalog == nil {
		return launchable, nil
	}
	entries, err := m.catalog.List()
	if err != nil {
		return nil, err
	}
	byCode := make(map[string]model.SystemCatalogEntry, len(entries))
	for _, e := range entries {
		byCode[e.SystemCode] = e
	}
	all, err := m.systems.GetSystemsByClientId(user.ClientId)
	if err != nil {
		return nil, err
	}
	for _, s := range all {
		entry, listed := byCode[s.SystemCode]
		origin, ok := launchOrigin(entry.WebUrl)
		if !listed || !ok || s.SystemCode == current || !systemAllowed(user.Role, s.SystemCode) {
			continue
		}
		s.SystemName, s.WebUrl = entry.SystemName, origin
		launchable = append(launchable, s)
	}
	return launchable, nil
}

// Admit checks a User may start a Session on a System, by the rules every
// login shares, before an SSO ticket is issued for it.
func (m *Manager) Admit(userId, system string) error {
	if m.systems == nil {
		return errors.New("session manager has no login (WithLogin)")
	}
	user, err := m.users.GetUserById(userId)
	if err != nil || user == nil {
		return ErrSessionInvalid
	}
	return m.eligible(user, system)
}

// launchOrigin accepts an https origin, or http on a loopback host for local
// development, and drops any path, query or credentials.
func launchOrigin(webUrl string) (string, bool) {
	u, err := url.Parse(webUrl)
	if err != nil || u.Hostname() == "" || u.User != nil {
		return "", false
	}
	loopback := u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"
	if u.Scheme != "https" && !(u.Scheme == "http" && loopback) {
		return "", false
	}
	return u.Scheme + "://" + u.Host, true
}
