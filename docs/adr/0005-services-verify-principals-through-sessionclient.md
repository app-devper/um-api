# Services verify Principals through sessionclient

Extends [ADR-0004](./0004-services-read-sessions-through-sessionclient.md) and replaces two of its clauses.

Every service that accepts UM tokens used to rebuild the same pipeline: parse the JWT, bind it to its system and optionally its client, confirm the session, map failures to 401/503, and store the caller under untyped keys. The copies drifted: gold-shop never bound the token to its system, snook skipped the signing-method check, alert admitted a `STAFF` role UM never issues, and 503 codes differed per service. `sessionclient` now owns the whole pipeline: `Verifier.Verify` turns a request into a **Principal** (CONTEXT.md) and every service goes through it, with a thin adapter per router (`sessionclient/ginauth` for gin; chi uses the net/http middleware directly). UM's own routes keep using UM's session module, which owns the store.

**Refusals** use one code set everywhere: `AU-401-001` missing token, `AU-401-002` invalid token, `AU-401-003` another system, `AU-401-004` another client, `AU-401-005` session invalid, `AU-403-002` role too low, `AU-503-001` store unavailable. The 503 message is always `identity service unavailable`; pharmacy clients match on it to keep work pending instead of signing out. Each service keeps its own response envelope.

**Roles** follow UM's ordering `USER < MANAGER < ADMIN < SUPER`, exported by sessionclient as `Role.AtLeast`; UM pins that its role constants match. A route names its minimum role instead of listing roles, so an unknown role reaches nothing and no role can be skipped by accident.

**Replaced from ADR-0004:**

- *"Unset, it keeps its previous behaviour."* A service without UM's session store now refuses to start (`NewVerifier` requires a store; `RedisStoreFor("")` is an error). Tests use a fake store. Each service keeps its existing environment variable for the Redis address.
- *One outage policy for everyone.* The **Outage policy** is chosen per route group: `ReadOnlyWithLastGood` (default: safe methods continue with the last session confirmed for the token), `Strict` (writes and sensitive reads), `DegradeReads` (ordinary catalog reads continue under the signed token). This absorbs the pharmacy API's per-route policy from its ADR-0001 without changing it.

Every service moved in sessionclient v0.2.x; v0.3.0 removed the older `Checker.Check`/`Authorize` API so the Verifier is the only way in.
