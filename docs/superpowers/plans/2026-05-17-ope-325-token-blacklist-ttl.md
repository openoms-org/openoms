# OPE-325: Token blacklist TTL from access token expiry

## Goal and Scope

Fix logout access-token revocation so the blacklist entry expires at the JWT's actual `exp` timestamp instead of always using `now + 1h`.

Scope is limited to the public API auth/logout path:
- `apps/api-server/internal/handler/auth_handler.go`
- `apps/api-server/internal/handler/auth_handler_test.go`
- `apps/api-server/internal/service/auth_service.go` if a small token-validation helper is needed
- `.claude/context/SECURITY_POSTURE.md`

Out of scope:
- broader JWT middleware ordering from OPE-333,
- CSRF logout behavior from OPE-322,
- refresh token rotation internals.

## Implementation Tasks

- [x] Add a regression test proving logout blacklists a valid access token until the token's own expiry.
- [x] Add a regression test proving malformed/invalid access tokens are not added to the blacklist.
- [x] Implement a small validated-access-token expiry helper, avoiding unverified JWT parsing.
- [x] Replace the hardcoded `time.Now().Add(1*time.Hour)` in logout with the validated token expiry.
- [x] Keep logout idempotent: token expiry validation failures must not prevent refresh-cookie clearing.
- [x] Update security posture documentation.

## Test and Validation Plan

- [x] Red: targeted handler test fails against current hardcoded TTL behavior.
- [x] Green: `go test ./internal/handler -run 'TestAuthHandler_Logout.*Blacklist' -count=1`
- [x] Targeted auth package tests: `go test ./internal/handler ./internal/service ./internal/middleware -count=1`
- [x] `git diff --check`
- [x] Full public repo validation before push: `./scripts/local-ci.sh`

## Risks and Rollback

Risk is low but security-sensitive. Logout is a public route and must remain best-effort/idempotent, so invalid/missing bearer tokens should not change the HTTP response.

Rollback is simple: revert the PR. The previous behavior blacklisted for `now + 1h`, which is safe for freshly issued tokens but incorrect for near-expiry tokens and unnecessarily long for short-lived typed tokens.
