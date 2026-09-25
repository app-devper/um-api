# Service session verification

_Partly superseded by [ADR-0003](./0003-pharmacy-reads-um-sessions-from-redis.md): the pharmacy API now reads UM's Redis directly._

UM revokes sessions and reloads current user state on its own protected requests, but a pharmacy service that checks only a signed token can continue using an obsolete role until token expiry. UM will own a dedicated contract for another service to verify the current session, user status, role, system, and client/tenant without sharing Redis or MongoDB. The pharmacy service must bind that result to the signed token's identity, system, and tenant and may cache it for at most 30 seconds; after the last valid check expires, it stops confirming writes and sensitive reads if UM is unavailable. The cross-service revocation bound is 60 seconds.
