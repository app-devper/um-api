package model

import "time"

// Passkey is a WebAuthn credential a User registered (ADR-0010). Credential
// holds the library's credential record as JSON so its shape can change
// without a migration.
type Passkey struct {
	Id           string    `bson:"_id" json:"id"`
	UserId       string    `bson:"userId" json:"-"`
	Name         string    `bson:"name" json:"name"`
	Credential   []byte    `bson:"credential" json:"-"`
	CreatedDate  time.Time `bson:"createdDate" json:"createdDate"`
	LastUsedDate time.Time `bson:"lastUsedDate,omitempty" json:"lastUsedDate,omitempty"`
}
