# ADR 0003 — JWT auth: short-lived access + rotating refresh family

- **Status:** accepted
- **Date:** 2026-05-07
- **Deciders:** backend agent (TASK_ID `micocards-mvp`)

## Context

The spec mandates email+password auth, no social login, no email verification for v1
(spec §Assumptions). The frontend (Reatom) needs a token it can carry on every
request without round-tripping a session lookup, and the server needs to be able to
revoke a stolen credential on the user's next legitimate refresh.

Constraints from `docs/stack.md`:

- Stdlib `net/http` + `golang-jwt/jwt/v5`.
- Postgres only (no Redis-backed session table).
- bcrypt for password hashing.

## Decision

Two-token scheme with explicit family-tracked rotation:

1. **Access token** — JWT, HS256, signed with `JWT_SECRET` (min 32 bytes, base64
   in `.env`). Lifetime: **15 minutes**. Claims: `sub` (user id), `iat`, `exp`,
   `jti` (random; reserved for future denylist). Stateless. Verified by
   `internal/interfaces/http/middleware/auth.go`. **No refresh-on-access**.
2. **Refresh token** — opaque random 256-bit string (`base64url(32 bytes from
   crypto/rand)`). Lifetime: **7 days**. Persisted in `iam.refresh_tokens` with
   columns: `id, user_id, family_id, parent_id, opaque_hash, issued_at,
   expires_at, revoked_at, revoke_reason`. Stored as a SHA-256 hash, never
   plaintext.
3. **Rotation.** Every successful `POST /api/auth/refresh`:
   a. Verifies the presented token belongs to a non-revoked family and is the
      latest token in that family (`parent_id` points at it from no other row).
   b. Inserts a new row in the same family with `parent_id = consumed.id`.
   c. Sets `revoked_at` on the consumed row with reason `rotation`.
   d. Mints and returns a new access + new refresh.
4. **Reuse detection.** If `RefreshAccessToken` is called with a token that **is
   already revoked** (i.e. someone replayed a leaked older token), the entire
   family is revoked (`RevokeFamily`) and the call returns `ErrRefreshTokenReused`
   (HTTP 401). The legitimate user must log in again. This is the only mechanism
   for compromised-token containment.
5. **`ChangePassword`** revokes every active family for the user.
6. **`LogoutUser`** revokes the family of the supplied refresh token.

## Alternatives considered

- **Long-lived access tokens, no refresh.** Cannot revoke a leaked token without a
  denylist (which then becomes the session table we wanted to avoid). Rejected.
- **Sliding-session cookies.** Would require a server-side session table and pin
  the auth model to cookies. Rejected — Reatom and React Native PWA install both
  prefer Authorization-header bearer tokens.
- **OAuth2 with a third-party IdP.** Spec excludes social login.
- **JWT for refresh too.** Self-revocable JWTs require a denylist for rotation,
  which is the same as our table — but harder to query. Opaque tokens are simpler.

## Consequences

**Positive.**
- Stateless access checks — middleware is pure crypto + clock.
- Clean compromise containment via family revocation.
- Predictable token lifetime: a stolen access token is dead in ≤15 min; a stolen
  refresh token is dead the moment the legitimate user refreshes again.

**Negative.**
- Every refresh round-trips Postgres. Acceptable: refresh happens once per 15
  minutes per active client.
- Frontend must persist the refresh token securely. PWA stores it in
  `localStorage` (acceptable for the threat model; spec §Non-goals excludes
  social-login-grade hardening).

**Neutral.**
- `JWT_SECRET` rotation requires a downtime-tolerant strategy (publish new secret,
  accept both for a window, drop old). Out of scope for v1.
- Token denylist for individual access tokens is **not** implemented in v1; the
  short TTL is the substitute.
