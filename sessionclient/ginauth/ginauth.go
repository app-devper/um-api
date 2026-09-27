// Package ginauth adapts sessionclient's Verifier to gin.
//
// Require stores the verified Principal in the gin context and, for handlers
// written before Principal existed, the gin keys "SessionId", "UserId",
// "Role", "ClientId" and "System".
package ginauth

import (
	"github.com/app-devper/um-api/sessionclient"
	"github.com/gin-gonic/gin"
)

const principalKey = "Principal"

// Render writes a refusal in the service's own response shape and must abort
// the gin context.
type Render func(c *gin.Context, e *sessionclient.Error)

type Auth struct {
	verifier *sessionclient.Verifier
	render   Render
}

func New(verifier *sessionclient.Verifier, render Render) *Auth {
	return &Auth{verifier: verifier, render: render}
}

// Require verifies the request's token and live session under policy.
func (a *Auth) Require(policy sessionclient.OutagePolicy) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := a.verifier.Verify(c.Request, policy)
		if err != nil {
			a.render(c, err.(*sessionclient.Error))
			c.Abort()
			return
		}
		c.Set(principalKey, p)
		c.Set("SessionId", p.SessionID)
		c.Set("UserId", p.UserID)
		c.Set("Role", string(p.Role))
		c.Set("ClientId", p.ClientID)
		c.Set("System", p.System)
		c.Request = c.Request.WithContext(sessionclient.WithPrincipal(c.Request.Context(), p))
		c.Next()
	}
}

// AtLeast refuses a Principal whose role is below min. It must run after
// Require.
func (a *Auth) AtLeast(min sessionclient.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := Principal(c)
		if !ok || !p.Role.AtLeast(min) {
			a.render(c, sessionclient.Forbidden())
			c.Abort()
			return
		}
		c.Next()
	}
}

// Principal returns the Principal Require stored in c.
func Principal(c *gin.Context) (sessionclient.Principal, bool) {
	v, ok := c.Get(principalKey)
	if !ok {
		return sessionclient.Principal{}, false
	}
	p, ok := v.(sessionclient.Principal)
	return p, ok
}
