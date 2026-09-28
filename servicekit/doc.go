// Package servicekit holds the plumbing every devper service needs and used
// to copy (um-api ADR-0007): the gateway-origin check (package gateway) and the
// database-per-tenant registry (package tenant). It depends on no web
// framework or database driver; each service keeps its own error envelope
// and seeding.
package servicekit
