// Package servicekit holds the plumbing every devper service needs and used
// to copy (um-api ADR-0007): the gateway-origin check (package gateway, with
// a gin adapter in gateway/gingateway) and the database-per-tenant registry
// (package tenant). Each service keeps its own error envelope and seeding.
package servicekit
