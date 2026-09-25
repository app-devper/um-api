# UM API architecture notes

The [product-wide context map](https://github.com/app-devper/pharmacy-app-kmp/blob/develop/CONTEXT-MAP.md) defines pharmacy business ownership. UM owns user identity, sessions, role assignment, and system access. Its [OpenAPI](./openapi.yaml) remains the endpoint contract for this service.

## Current behavior and agreed target

| Area | Current code | Agreed target |
| --- | --- | --- |
| Login and session | [auth use case](../app/domain/usecase/auth.go) issues a 24-hour JWT and stores a session. Every UM protected request checks the session and reloads the user; role/status changes revoke sessions. | Preserve immediate revocation within UM. Provide a dedicated verification contract for pharmacy operations without exposing UM Redis or MongoDB to the pharmacy API. |
| Pharmacy verification | Per [ADR-0003](./adr/0003-pharmacy-reads-um-sessions-from-redis.md) the pharmacy API reads `session:<jti>` from UM's Redis (format pinned in [redis_test.go](../app/domain/repository/redis_test.go)). UM revokes sessions on every role, status, password, or deletion change in the [user administration module](../app/domain/useradmin/useradmin.go), so a live session vouches for its token's claims. `GET /auth/verify` remains for services without Redis access. | Return current session, user, role, system, and client/tenant status for a signed token. The pharmacy API binds the response to the same identity, system, and tenant and caches it for at most 30 seconds; pharmacy writes and sensitive reads reflect revocation within 60 seconds. |
| Roles | UM supports SUPER, ADMIN, MANAGER, and USER. MANAGER can list/get users in its own client but cannot mutate them. | Keep these identity-service rules. The pharmacy API separately grants MANAGER operational stock, receipt, supplier, customer-edit, label, and slow-drug-read permissions; financial/KY/void/store-setting administration stays ADMIN+. |
| Tenant scope | SUPER is reserved for client `000` and can administer users across clients. | This UM capability does not grant implicit access to another pharmacy tenant's business data. Any future pharmacy support delegation must name the tenant and leave an audit trail. |
| Cross-repository checks | UM has OpenAPI and local Go CI, but no check that KMP and pharmacy API still satisfy the shared identity contract. | Check UM route/response drift on affected PRs, verify KMP's used endpoints/DTOs, and join a three-service checkout/authorization smoke test nightly and before release. |

The current [pharmacy API](https://github.com/app-devper/pharmacy-api/blob/develop/middleware/auth.go) trusts JWT role claims until token expiry instead of checking the live UM session. The pharmacy API reads UM sessions from Redis with a 30-second cache (ADR-0003).
