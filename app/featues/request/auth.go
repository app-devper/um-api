package request

import "encoding/json"

type Login struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	System   string `json:"system" binding:"required"`
}

// SSOTicket optionally names the System the ticket opens (ADR-0011).
type SSOTicket struct {
	System string `json:"system" binding:"omitempty,max=50"`
}

type ExchangeTicket struct {
	Ticket string `json:"ticket" binding:"required"`
}

type PasskeyLoginOptions struct {
	System string `json:"system" binding:"required"`
}

type PasskeyLogin struct {
	CeremonyId string          `json:"ceremonyId" binding:"required"`
	Credential json.RawMessage `json:"credential" binding:"required"`
}

type PasskeyRegistrationOptions struct {
	Password string `json:"password" binding:"required"`
}

type PasskeyRegistration struct {
	CeremonyId string          `json:"ceremonyId" binding:"required"`
	Name       string          `json:"name"`
	Credential json.RawMessage `json:"credential" binding:"required"`
}
