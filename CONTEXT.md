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
The user performing an administrative operation on another user, judged by their current role and client. An actor may only manage users whose role it outranks, within its own client unless it is a super user.
_Avoid_: Caller, requester

**Role**:
A user's permission tier, checked against the user's current account state when an operation is authorized.
_Avoid_: Token claim

**Super user**:
A role reserved for client `000` that may administer users across clients. It does not by itself select another tenant's pharmacy data.
_Avoid_: Implicit pharmacy tenant access
