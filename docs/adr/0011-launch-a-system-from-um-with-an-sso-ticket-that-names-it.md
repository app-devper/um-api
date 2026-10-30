# Launch a System from UM with an SSO ticket that names it

Users signed in to UM asked to open their other Systems (PortForMe, Pharmacy) from there without signing in again. SSO tickets already moved a User from another app into UM's profile page, on the same System as the Session that asked.

Decisions:

- **A ticket may name the System it opens.** `POST /auth/sso-ticket` takes an optional `system`. When it differs from the caller's System, `session.Manager.Admit` applies the login rules for that System — ACTIVE, the User's Client has it, SUPER only in SM — and refuses with 403 `UM-403-002` otherwise. Without `system` the ticket keeps the caller's System, as before.
- **Exchange re-checks the full login rules (ADR-0009).** `Resume` now also checks the User's Client still has the ticket's System, so a ticket cannot outlive a System being removed from the Client during its 60 seconds.
- **A System records where its front end lives.** `host` already holds a System's API address (GOLD, SNOOK and ALERT call their API there), so a new optional `webUrl` holds the front end's address. `GET /auth/systems` returns the other Systems of the caller's Client that the caller may use and whose `webUrl` is an https origin (http only on localhost). Each `webUrl` is reduced to its origin, so the launch link is always `<origin>/sso?ticket=…` and a stored path or query cannot redirect elsewhere.
- **The target app owns its Session.** The target exchanges the ticket for its own Session on its own System; signing out of UM does not sign out of it. An app that proxies login (PortForMe) exchanges through its API and accepts only tokens for its own System.

Consequences: a System appears in the launcher only once a SUPER sets its `webUrl` and its front end serves `/sso?ticket=`; Systems without a web front end (POS today) stay off the list. Tickets still travel in the URL for at most 60 seconds and are consumed once.
