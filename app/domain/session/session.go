// Package session owns the Session lifecycle: issuing a signed access token
// bound to a stored Session, verifying a token against the live Session and
// User, and renewing it. Callers never see JWT claims or the session store.
package session

import (
	"errors"
	"fmt"
	"time"
	"um/app/core/config"
	"um/app/core/constant"
	"um/app/domain/model"
	"um/app/domain/repository"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrNoSecret means the signing key is not configured.
	ErrNoSecret = errors.New("SECRET_KEY is not set")
	// ErrTokenInvalid means the token is malformed, badly signed, or expired.
	ErrTokenInvalid = errors.New("token invalid")
	// ErrSessionInvalid means the Session was revoked or its User no longer exists.
	ErrSessionInvalid = errors.New("session invalid")
	// ErrUserInactive means the Session's User is no longer ACTIVE.
	ErrUserInactive = errors.New("user inactive")
	// ErrTokenGenFailed means signing failed; the Session has already been removed.
	ErrTokenGenFailed = errors.New("failed to generate token")
)

// Principal is the verified caller behind a token. Role and ClientId come from
// the User's current account state, not from the token.
type Principal struct {
	SessionId string
	UserId    string
	Role      string
	System    string
	ClientId  string
}

// Metadata describes where a Session was started.
type Metadata struct {
	UserAgent string
	IPAddress string
}

type Manager struct {
	secretKey []byte
	sessions  repository.ISession
	users     repository.IUser
	ttl       time.Duration
}

func NewManager(secretKey string, sessions repository.ISession, users repository.IUser) *Manager {
	return &Manager{
		secretKey: []byte(secretKey),
		sessions:  sessions,
		users:     users,
		ttl:       config.AccessTokenTime,
	}
}

// Issue starts a Session for an authenticated, ACTIVE user on a System and
// returns its access token. No Session is left behind if signing fails.
func (m *Manager) Issue(user *model.User, system string, meta Metadata) (string, error) {
	userId := user.Id.Hex()
	sessionId, err := m.sessions.CreateSession(userId, m.ttl, repository.SessionMetadata{
		UserAgent: meta.UserAgent,
		IPAddress: meta.IPAddress,
		System:    system,
	})
	if err != nil {
		return "", err
	}
	return m.sign(Principal{
		SessionId: sessionId,
		UserId:    userId,
		Role:      user.Role,
		System:    system,
		ClientId:  user.ClientId,
	})
}

// Verify checks the token signature, then the live Session and User.
func (m *Manager) Verify(token string) (*Principal, error) {
	if len(m.secretKey) == 0 {
		return nil, ErrNoSecret
	}
	claims := &accessClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secretKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}
	if parsed == nil || !parsed.Valid || claims.ID == "" {
		return nil, ErrTokenInvalid
	}

	stored, err := m.sessions.GetSessionById(claims.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSessionInvalid, err)
	}
	// Sessions created before System was recorded carry none; accept those.
	if stored.System != "" && stored.System != claims.System {
		return nil, fmt.Errorf("%w: token system %q does not match session", ErrSessionInvalid, claims.System)
	}
	userId := stored.UserId
	user, err := m.users.GetUserById(userId)
	if err != nil || user == nil {
		return nil, ErrSessionInvalid
	}
	if user.Status != constant.ACTIVE {
		return nil, ErrUserInactive
	}
	return &Principal{
		SessionId: claims.ID,
		UserId:    userId,
		Role:      user.Role,
		System:    claims.System,
		ClientId:  user.ClientId,
	}, nil
}

// Renew extends a verified Session and returns a fresh token. If signing
// fails the Session is removed.
func (m *Manager) Renew(p *Principal) (string, error) {
	if err := m.sessions.UpdateSessionExpireById(p.SessionId, m.ttl); err != nil {
		return "", err
	}
	return m.sign(*p)
}

type accessClaims struct {
	Role     string `json:"role"`
	System   string `json:"system"`
	ClientId string `json:"clientId"`
	jwt.RegisteredClaims
}

func (m *Manager) sign(p Principal) (string, error) {
	token, err := m.signJwt(p)
	if err != nil {
		if removeErr := m.sessions.RemoveSessionById(p.SessionId); removeErr != nil {
			return "", fmt.Errorf("%w: %v (remove session: %v)", ErrTokenGenFailed, err, removeErr)
		}
		return "", fmt.Errorf("%w: %v", ErrTokenGenFailed, err)
	}
	return token, nil
}

func (m *Manager) signJwt(p Principal) (string, error) {
	if len(m.secretKey) == 0 {
		return "", ErrNoSecret
	}
	claims := &accessClaims{
		Role:     p.Role,
		System:   p.System,
		ClientId: p.ClientId,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        p.SessionId,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secretKey)
}
