package passkey

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/go-webauthn/webauthn/protocol/webauthncbor"
)

const (
	flagUserPresent  = 0x01
	flagUserVerified = 0x04
	flagAttested     = 0x40
)

// softAuthenticator is a platform authenticator in software: it holds one
// ES256 key and answers creation and assertion Ceremonies the way a browser
// would hand them to UM, so the tests exercise real WebAuthn verification.
type softAuthenticator struct {
	t          *testing.T
	rpID       string
	origin     string
	key        *ecdsa.PrivateKey
	credID     []byte
	userHandle []byte
	signCount  uint32
	skipUV     bool
}

func newSoftAuthenticator(t *testing.T, rpID, origin string) *softAuthenticator {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	credID := make([]byte, 16)
	_, _ = rand.Read(credID)
	return &softAuthenticator{t: t, rpID: rpID, origin: origin, key: key, credID: credID}
}

type publicKeyOptions struct {
	PublicKey struct {
		Challenge string `json:"challenge"`
		User      struct {
			ID string `json:"id"`
		} `json:"user"`
	} `json:"publicKey"`
}

func (a *softAuthenticator) parse(options []byte) publicKeyOptions {
	a.t.Helper()
	var o publicKeyOptions
	if err := json.Unmarshal(options, &o); err != nil {
		a.t.Fatal(err)
	}
	return o
}

func (a *softAuthenticator) clientData(kind, challenge string) []byte {
	data, _ := json.Marshal(map[string]any{"type": kind, "challenge": challenge, "origin": a.origin, "crossOrigin": false})
	return data
}

func (a *softAuthenticator) authData(flags byte, attested []byte) []byte {
	rpHash := sha256.Sum256([]byte(a.rpID))
	data := append([]byte{}, rpHash[:]...)
	data = append(data, flags)
	count := make([]byte, 4)
	binary.BigEndian.PutUint32(count, a.signCount)
	data = append(data, count...)
	return append(data, attested...)
}

func (a *softAuthenticator) cosePublicKey() []byte {
	x := make([]byte, 32)
	y := make([]byte, 32)
	a.key.PublicKey.X.FillBytes(x)
	a.key.PublicKey.Y.FillBytes(y)
	encoded, err := webauthncbor.Marshal(map[int]any{1: 2, 3: -7, -1: 1, -2: x, -3: y})
	if err != nil {
		a.t.Fatal(err)
	}
	return encoded
}

// create answers registration options with a "none" attestation.
func (a *softAuthenticator) create(options []byte) []byte {
	a.t.Helper()
	o := a.parse(options)
	userHandle, err := b64.DecodeString(o.PublicKey.User.ID)
	if err != nil {
		a.t.Fatal(err)
	}
	a.userHandle = userHandle
	attested := make([]byte, 16)
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(a.credID)))
	attested = append(attested, length...)
	attested = append(attested, a.credID...)
	attested = append(attested, a.cosePublicKey()...)
	attestation, err := webauthncbor.Marshal(map[string]any{
		"fmt":      "none",
		"attStmt":  map[string]any{},
		"authData": a.authData(flagUserPresent|flagUserVerified|flagAttested, attested),
	})
	if err != nil {
		a.t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{
		"id":    b64.EncodeToString(a.credID),
		"rawId": b64.EncodeToString(a.credID),
		"type":  "public-key",
		"response": map[string]any{
			"clientDataJSON":    b64.EncodeToString(a.clientData("webauthn.create", o.PublicKey.Challenge)),
			"attestationObject": b64.EncodeToString(attestation),
		},
		"clientExtensionResults": map[string]any{},
	})
	return body
}

// get answers login options with a signed assertion.
func (a *softAuthenticator) get(options []byte) []byte {
	a.t.Helper()
	o := a.parse(options)
	clientData := a.clientData("webauthn.get", o.PublicKey.Challenge)
	flags := byte(flagUserPresent | flagUserVerified)
	if a.skipUV {
		flags = flagUserPresent
	}
	authData := a.authData(flags, nil)
	clientHash := sha256.Sum256(clientData)
	digest := sha256.Sum256(append(append([]byte{}, authData...), clientHash[:]...))
	signature, err := ecdsa.SignASN1(rand.Reader, a.key, digest[:])
	if err != nil {
		a.t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{
		"id":    b64.EncodeToString(a.credID),
		"rawId": b64.EncodeToString(a.credID),
		"type":  "public-key",
		"response": map[string]any{
			"clientDataJSON":    b64.EncodeToString(clientData),
			"authenticatorData": b64.EncodeToString(authData),
			"signature":         b64.EncodeToString(signature),
			"userHandle":        b64.EncodeToString(a.userHandle),
		},
		"clientExtensionResults": map[string]any{},
	})
	return body
}
