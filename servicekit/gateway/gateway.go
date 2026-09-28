// Package gateway refuses requests that did not come through the api.devper.app
// gateway (Firebase Hosting rewrites), by the X-Forwarded-Host it sets.
//
// The header can be forged by a direct caller; this keeps casual direct access
// to a service's Cloud Run URL out, it is not an authentication boundary
// (um-api ADR-0007). With no hosts configured every request is allowed.
package gateway

import (
	"net/http"
	"strings"
)

// Hosts is the set of gateway hosts a service accepts, from a
// comma-separated list such as the GATEWAY_HOSTS environment variable.
type Hosts struct{ allowed map[string]bool }

// ParseHosts reads a comma-separated host list; blanks are ignored.
func ParseHosts(list string) Hosts {
	h := Hosts{allowed: map[string]bool{}}
	for _, host := range strings.Split(list, ",") {
		if host = strings.ToLower(strings.TrimSpace(host)); host != "" {
			h.allowed[host] = true
		}
	}
	return h
}

// Enabled reports whether any host is configured.
func (h Hosts) Enabled() bool { return len(h.allowed) > 0 }

// Allows reports whether r came through one of the hosts (always, when none
// is configured).
func (h Hosts) Allows(r *http.Request) bool {
	if !h.Enabled() {
		return true
	}
	forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0])
	return h.allowed[strings.ToLower(forwarded)]
}

// Message is the reason given for a refused request.
const Message = "direct access is not allowed"

// Middleware passes requests the hosts allow and hands the rest to refuse,
// which writes the service's own 403 response.
func Middleware(h Hosts, refuse http.HandlerFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if !h.Enabled() {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !h.Allows(r) {
				refuse(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
