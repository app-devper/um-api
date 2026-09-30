# Identity and Access

This context owns the people, sessions, roles, and system access used by the pharmacy app and other client applications.

## Language

**User**:
A person with an account, status, and role within a client organization.
_Avoid_: Pharmacy customer

**Client**:
An organization identified by `clientId` whose users and system access are managed separately from other clients.
_Avoid_: System, user

**System**:
An application a client is permitted to access. It is distinct from the client organization and from a login session.
_Avoid_: Client, session

**Session**:
An active authenticated relationship between a user and a system that can be revoked independently of an access token's expiry.
_Avoid_: Token

**Principal**:
The verified caller behind an access token: its session, user, system, and the user's current role and client. Produced only by verifying a token against the live session and user.
_Avoid_: Claims, token payload

**Actor**:
The user performing an administrative operation on another user, judged by their current role and client. An actor may only manage users whose role it outranks, within its own client unless it is a super user. UM reports what an actor may do to each user it lists (`can`) and which roles it may create, so clients do not re-derive role order (ADR-0006).
_Avoid_: Caller, requester

**Role**:
A user's permission tier, checked against the user's current account state when an operation is authorized. SUPER belongs only to client 000 and is used only in System SM (user management); it never reaches another System such as POS, pharmacy, snook, gold or alert.
_Avoid_: Token claim

**Outage policy**:
What a request may do while UM's session store cannot answer, chosen per group of routes: continue a read with the last confirmed session, refuse everything, or continue ordinary reads under the signed token. A revoked or expired session is refused under every policy.
_Avoid_: Fallback, fail-open

**Super user**:
A role reserved for client `000` that may administer users across clients. It does not by itself select another tenant's pharmacy data.
_Avoid_: Implicit pharmacy tenant access
