// Package passkey owns Passkeys (ADR-0010): registering a WebAuthn
// credential for a User, using one to start a Session, and listing and
// deleting a User's own Passkeys. Starting the Session itself stays with the
// session module, which applies the login rules.
package passkey

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"um/app/core/utils"
	"um/app/domain/model"
	"um/app/domain/repository"
	"um/app/domain/session"
	"unicode/utf8"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

var (
	// ErrDisabled: UM has no relying party configured (WEBAUTHN_RP_ID unset).
	ErrDisabled = errors.New("passkeys are not enabled")
	// ErrWrongPassword: registering a Passkey needs the User's current password.
	ErrWrongPassword = errors.New("wrong password")
	// ErrCeremony: the Ceremony expired, was already answered, or belongs to someone else.
	ErrCeremony = errors.New("passkey request expired, start again")
	// ErrNotAccepted: the authenticator's answer did not verify.
	ErrNotAccepted = errors.New("passkey not accepted")
	// ErrOtherAccount: the passkey verified but belongs to a different User
	// than the username typed on the login page.
	ErrOtherAccount = errors.New("this passkey belongs to a different account")
	// ErrNotFound: the User has no such Passkey.
	ErrNotFound = errors.New("passkey not found")
)

const (
	kindRegister   = "register"
	kindLogin      = "login"
	defaultName    = "Passkey"
	maxNameLength  = 60
	userHandleSize = 24
)

var b64 = base64.RawURLEncoding

// Config is the relying party: the domain Passkeys belong to and the page
// origins allowed to run a Ceremony.
type Config struct {
	RPID    string
	RPName  string
	Origins []string
}

// Options is what a front end passes to navigator.credentials.create/get,
// with the Ceremony it must answer.
type Options struct {
	CeremonyId string          `json:"ceremonyId"`
	PublicKey  json.RawMessage `json:"options"`
}

type Service struct {
	wa         *webauthn.WebAuthn
	keys       repository.IPasskey
	ceremonies repository.IPasskeyCeremony
	users      repository.IUser
	sessions   *session.Manager
	now        func() time.Time
}

// New builds the Service. An empty RPID gives a Service whose every call
// answers ErrDisabled.
func New(cfg Config, keys repository.IPasskey, ceremonies repository.IPasskeyCeremony, users repository.IUser, sessions *session.Manager) (*Service, error) {
	s := &Service{keys: keys, ceremonies: ceremonies, users: users, sessions: sessions, now: time.Now}
	if cfg.RPID == "" {
		return s, nil
	}
	name := cfg.RPName
	if name == "" {
		name = cfg.RPID
	}
	wa, err := webauthn.New(&webauthn.Config{RPID: cfg.RPID, RPDisplayName: name, RPOrigins: cfg.Origins})
	if err != nil {
		return nil, fmt.Errorf("webauthn config: %w", err)
	}
	s.wa = wa
	return s, nil
}

// Enabled reports whether a relying party is configured.
func (s *Service) Enabled() bool { return s.wa != nil }

type registrationState struct {
	UserId  string               `json:"userId"`
	Session webauthn.SessionData `json:"session"`
}

type loginState struct {
	System  string               `json:"system"`
	Session webauthn.SessionData `json:"session"`
}

// BeginRegistration starts adding a Passkey for a User who re-enters their
// password, so a stolen access token alone cannot add one.
func (s *Service) BeginRegistration(userId, password string) (Options, error) {
	if !s.Enabled() {
		return Options{}, ErrDisabled
	}
	user, err := s.users.GetUserById(userId)
	if err != nil || user == nil {
		return Options{}, ErrWrongPassword
	}
	if utils.ComparePasswordAndHashedPassword(password, user.Password) != nil {
		return Options{}, ErrWrongPassword
	}
	acct, err := s.account(user)
	if err != nil {
		return Options{}, err
	}
	creation, sd, err := s.wa.BeginRegistration(acct,
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			ResidentKey:        protocol.ResidentKeyRequirementRequired,
			RequireResidentKey: protocol.ResidentKeyRequired(),
			UserVerification:   protocol.VerificationRequired,
		}),
		webauthn.WithExclusions(webauthn.Credentials(acct.credentials).CredentialDescriptors()),
	)
	if err != nil {
		return Options{}, err
	}
	return s.save(kindRegister, registrationState{UserId: userId, Session: *sd}, creation)
}

// FinishRegistration verifies the authenticator's answer and stores the Passkey.
func (s *Service) FinishRegistration(userId, ceremonyId, name string, response []byte) (model.Passkey, error) {
	if !s.Enabled() {
		return model.Passkey{}, ErrDisabled
	}
	var state registrationState
	if err := s.consume(kindRegister, ceremonyId, &state); err != nil {
		return model.Passkey{}, err
	}
	if state.UserId != userId {
		return model.Passkey{}, ErrCeremony
	}
	user, err := s.users.GetUserById(userId)
	if err != nil || user == nil {
		return model.Passkey{}, ErrCeremony
	}
	acct, err := s.account(user)
	if err != nil {
		return model.Passkey{}, err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(response)
	if err != nil {
		return model.Passkey{}, fmt.Errorf("%w: %v", ErrNotAccepted, err)
	}
	credential, err := s.wa.CreateCredential(acct, state.Session, parsed)
	if err != nil {
		return model.Passkey{}, fmt.Errorf("%w: %v", ErrNotAccepted, err)
	}
	stored, err := json.Marshal(credential)
	if err != nil {
		return model.Passkey{}, err
	}
	key := model.Passkey{
		Id:          b64.EncodeToString(credential.ID),
		UserId:      userId,
		Name:        cleanName(name),
		Credential:  stored,
		CreatedDate: s.now().UTC(),
	}
	if err := s.keys.Create(key); err != nil {
		return model.Passkey{}, err
	}
	return key, nil
}

// BeginLogin starts a username-less login on a System: the authenticator
// offers the Passkeys it holds for this relying party.
func (s *Service) BeginLogin(system string) (Options, error) {
	if !s.Enabled() {
		return Options{}, ErrDisabled
	}
	assertion, sd, err := s.wa.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		return Options{}, err
	}
	return s.save(kindLogin, loginState{System: system, Session: *sd}, assertion)
}

// FinishLogin verifies the assertion, records the new sign count, and starts
// a Session through the session module, which applies the login rules. A
// non-empty username is the one typed on the login page: the passkey's owner
// must be that User (ADR-0010).
func (s *Service) FinishLogin(ceremonyId string, response []byte, username string, meta session.Metadata) (string, error) {
	if !s.Enabled() {
		return "", ErrDisabled
	}
	var state loginState
	if err := s.consume(kindLogin, ceremonyId, &state); err != nil {
		return "", err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(response)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrNotAccepted, err)
	}
	var key *model.Passkey
	var owner *model.User
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		found, err := s.keys.Get(b64.EncodeToString(rawID))
		if err != nil {
			return nil, err
		}
		if found.UserId != string(userHandle) {
			return nil, errors.New("credential belongs to another user")
		}
		user, err := s.users.GetUserById(found.UserId)
		if err != nil || user == nil {
			return nil, errors.New("user not found")
		}
		credential, err := decode(found.Credential)
		if err != nil {
			return nil, err
		}
		key = found
		owner = user
		return &account{user: user, credentials: []webauthn.Credential{credential}}, nil
	}
	_, credential, err := s.wa.ValidatePasskeyLogin(handler, state.Session, parsed)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrNotAccepted, err)
	}
	if credential.Authenticator.CloneWarning {
		return "", fmt.Errorf("%w: sign count went backwards", ErrNotAccepted)
	}
	if typed := utils.NormalizeUsername(username); typed != "" && typed != owner.Username {
		return "", ErrOtherAccount
	}
	stored, err := json.Marshal(credential)
	if err != nil {
		return "", err
	}
	if err := s.keys.RecordUse(key.Id, stored, s.now().UTC()); err != nil {
		return "", err
	}
	return s.sessions.StartWithPasskey(key.UserId, state.System, meta)
}

// List returns a User's own Passkeys.
func (s *Service) List(userId string) ([]model.Passkey, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	return s.keys.ListByUser(userId)
}

// Delete removes one of the User's own Passkeys.
func (s *Service) Delete(userId, id string) error {
	if !s.Enabled() {
		return ErrDisabled
	}
	if err := s.keys.Delete(id, userId); err != nil {
		if errors.Is(err, repository.ErrPasskeyNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Service) save(kind string, state any, options any) (Options, error) {
	encodedState, err := json.Marshal(state)
	if err != nil {
		return Options{}, err
	}
	id, err := s.ceremonies.Save(kind, encodedState)
	if err != nil {
		return Options{}, err
	}
	encodedOptions, err := json.Marshal(options)
	if err != nil {
		return Options{}, err
	}
	return Options{CeremonyId: id, PublicKey: encodedOptions}, nil
}

func (s *Service) consume(kind, id string, state any) error {
	raw, err := s.ceremonies.Consume(kind, id)
	if errors.Is(err, repository.ErrCeremonyNotFound) {
		return ErrCeremony
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, state)
}

func (s *Service) account(user *model.User) (*account, error) {
	keys, err := s.keys.ListByUser(user.Id.Hex())
	if err != nil {
		return nil, err
	}
	credentials := make([]webauthn.Credential, 0, len(keys))
	for _, k := range keys {
		c, err := decode(k.Credential)
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, c)
	}
	return &account{user: user, credentials: credentials}, nil
}

func decode(raw []byte) (webauthn.Credential, error) {
	var c webauthn.Credential
	err := json.Unmarshal(raw, &c)
	return c, err
}

func cleanName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return defaultName
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		name = string([]rune(name)[:maxNameLength])
	}
	return name
}

// account is a User as WebAuthn sees them. The user handle is the User ID,
// which never changes and reveals nothing beyond an opaque identifier.
type account struct {
	user        *model.User
	credentials []webauthn.Credential
}

func (a *account) WebAuthnID() []byte { return []byte(a.user.Id.Hex())[:userHandleSize] }

func (a *account) WebAuthnName() string { return a.user.Username }

func (a *account) WebAuthnDisplayName() string {
	name := strings.TrimSpace(a.user.FirstName + " " + a.user.LastName)
	if name == "" {
		return a.user.Username
	}
	return name
}

func (a *account) WebAuthnCredentials() []webauthn.Credential { return a.credentials }
