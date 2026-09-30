package session

import (
	"errors"
	"fmt"
	"um/app/core/constant"
	"um/app/core/utils"
	"um/app/domain/model"
	"um/app/domain/repository"
)

// Why a Session could not be started.
var (
	// ErrWrongCredentials: unknown user or wrong password.
	ErrWrongCredentials = errors.New("wrong username or password")
	// ErrNotActive: the User is not ACTIVE.
	ErrNotActive = errors.New("account is not active")
	// ErrWrongSystem: the User's Client has no such System, or a SUPER asked
	// for a System other than SM.
	ErrWrongSystem = errors.New("invalid client or system")
	// ErrLocked: too many failed attempts.
	ErrLocked = errors.New("account locked due to too many failed login attempts, try again later")
	// ErrUnavailable: the lockout or session store cannot be reached; no
	// Session is started rather than skipping the lockout.
	ErrUnavailable = errors.New("identity store unavailable")
)

// Why Sessions could not be ended.
var (
	// ErrNotEnded: some Sessions are still live; ending them again is safe.
	ErrNotEnded = errors.New("sessions could not be ended")
	// ErrNotFound: no such Session.
	ErrNotFound = errors.New("session not found")
	// ErrNotOwner: the Session belongs to another User.
	ErrNotOwner = errors.New("session belongs to another user")
	// ErrCurrent: the current Session is ended by logging out.
	ErrCurrent = errors.New("cannot revoke current session, use logout instead")
)

// endAttempts is how often ending a User's Sessions is tried before failing.
const endAttempts = 3

// Credentials start a Session on a System.
type Credentials struct {
	Username string
	Password string
	System   string
}

// Start authenticates credentials and starts a Session: the User exists, the
// password matches, the User is ACTIVE, the User's Client has the System,
// and a SUPER uses only System SM. Every refusal but a lockout counts toward
// the lockout; success resets it.
func (m *Manager) Start(c Credentials, meta Metadata) (string, error) {
	if m.guard == nil || m.systems == nil {
		return "", errors.New("session manager has no login (WithLogin)")
	}
	username := utils.NormalizeUsername(c.Username)
	locked, err := m.guard.IsLocked(username)
	if err != nil {
		return "", fmt.Errorf("%w: lockout: %v", ErrUnavailable, err)
	}
	if locked {
		return "", ErrLocked
	}
	user, err := m.authenticate(username, c)
	if err != nil {
		_ = m.guard.RecordFailure(username)
		return "", err
	}
	_ = m.guard.Reset(username)
	return m.Issue(user, c.System, meta)
}

func (m *Manager) authenticate(username string, c Credentials) (*model.User, error) {
	user, err := m.users.GetUserByUsername(username)
	if err != nil || user == nil {
		return nil, ErrWrongCredentials
	}
	if utils.ComparePasswordAndHashedPassword(c.Password, user.Password) != nil {
		return nil, ErrWrongCredentials
	}
	if user.Status != constant.ACTIVE {
		return nil, ErrNotActive
	}
	if !systemAllowed(user.Role, c.System) {
		return nil, ErrWrongSystem
	}
	system, err := m.systems.GetSystem(user.ClientId, c.System)
	if err != nil || system == nil {
		return nil, ErrWrongSystem
	}
	return user, nil
}

// Resume starts a Session for a User who already holds one on the System,
// as an SSO ticket carries: the User must still be ACTIVE and allowed on it.
func (m *Manager) Resume(userId, system string, meta Metadata) (string, error) {
	user, err := m.users.GetUserById(userId)
	if err != nil || user == nil {
		return "", ErrSessionInvalid
	}
	if user.Status != constant.ACTIVE {
		return "", ErrUserInactive
	}
	if !systemAllowed(user.Role, system) {
		return "", ErrSessionInvalid
	}
	return m.Issue(user, system, meta)
}

// systemAllowed: SUPER uses only System SM (CONTEXT: Role).
func systemAllowed(role, system string) bool {
	return role != constant.SUPER || system == constant.SuperSystem
}

// End ends one Session (logout).
func (m *Manager) End(sessionId string) error {
	return m.sessions.RemoveSessionById(sessionId)
}

// EndOther ends another Session of the same User.
func (m *Manager) EndOther(userId, currentSessionId, targetId string) error {
	if targetId == currentSessionId {
		return ErrCurrent
	}
	target, err := m.sessions.GetSessionById(targetId)
	if err != nil {
		return ErrNotFound
	}
	if target.UserId != userId {
		return ErrNotOwner
	}
	return m.sessions.RemoveSessionById(targetId)
}

// EndAll ends every Session of a User except keep ("" ends all). It retries,
// and reports ErrNotEnded if any Session is still live, so the caller can
// fail its command instead of leaving services trusting old claims
// (ADR-0003).
func (m *Manager) EndAll(userId, keep string) (int, error) {
	total := 0
	var last error
	for i := 0; i < endAttempts; i++ {
		n, err := m.sessions.RevokeOtherSessions(userId, keep)
		total += n
		if err == nil {
			return total, nil
		}
		last = err
	}
	return total, fmt.Errorf("%w: %v", ErrNotEnded, last)
}

// List lists a User's live Sessions, marking the current one.
func (m *Manager) List(userId, currentSessionId string) ([]repository.SessionInfo, error) {
	return m.sessions.ListUserSessions(userId, currentSessionId)
}
