# The session module owns starting and ending Sessions

Login was a gin handler holding the whole policy: lockout, password, ACTIVE, System membership, and failure counting. SSO ticket creation and exchange repeated the ACTIVE check. Ending Sessions was split across the session handlers, `useradmin` and `session.Manager`. Three gaps followed:

- if the lockout store failed, login went ahead without the lockout;
- nothing kept a SUPER to System SM, although SUPER belongs to client 000 and is used only in SM (CONTEXT: Role);
- after a role, status, password change or a deletion, a failure to end the User's Sessions was only logged, and the command reported success. Other services treat a live Session as proof its claims are current (ADR-0003), so they kept authorising the old role until the token expired.

Now `session.Manager` owns both:

- `Start(credentials)` checks the lockout, then the user, password, ACTIVE, System membership, and SUPER only in SM. Every refusal but a lockout counts toward the lockout and success resets it. An unreachable lockout store refuses the login (503 `UM-503-001`); a Session cannot be issued without the same Redis anyway. `Resume` (SSO exchange) re-checks ACTIVE and the SUPER rule. `Verify` refuses a SUPER Session on any System but SM, so any that already exist stop working.
- `End`, `EndOther` and `EndAll` end Sessions. `EndAll` retries and reports any Session it could not end.

`useradmin` ends Sessions through it. A change whose Sessions could not all be ended answers 503 `UM-503-002`: the change is saved and repeating it is safe. A deletion ends the Sessions first, so a failed attempt leaves the User in place to retry. The auth and session handlers only translate HTTP.

**Addendum, 2026-11-07.** `ConfirmPassword` (used by `POST /auth/verify-password`) goes through the same lockout: a wrong confirmation counts toward the login counter, a right one resets it, and a locked User gets 429 `UM-429-001`. An unreachable lockout or user store refuses the confirmation (503 `UM-503-001`), as login does. The route had compared the hash itself with no guard, so a signed-in Session could guess the password without limit; this applies the decision above to a route it had missed. Two limits apply, and only the second is unconditional:

- the lockout is active only where `LOGIN_LOCKOUT_ENABLED` is set (it is off by default); with it off, `ConfirmPassword` counts nothing, exactly as login;
- the route carries a limiter per signed-in User (5 requests a minute, 429 `UM-429-001`), which holds whatever the lockout setting is and also closes the gap between the lockout's check and its count under parallel requests. It is keyed by User, not by address: the service sees every request from the gateway's address, so a limit per address would be one limit shared by all Users.

The shared counter cuts both ways: wrong confirmations from a Session lock that User's login, and failed logins for a username make that User's confirmations answer 429 until the lock expires or an ADMIN unlocks the User.
