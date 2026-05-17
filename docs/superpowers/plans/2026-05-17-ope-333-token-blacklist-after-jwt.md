# OPE-333 Token Blacklist After JWT Validation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Avoid querying token blacklist storage for malformed, expired, or otherwise invalid JWTs.

**Architecture:** Keep `JWTAuth` as the only behavior boundary. Validate the token signature and claims first, reject invalid tokens immediately, then hash/check the blacklist only for validated access-token candidates.

**Tech Stack:** Go, `net/http/httptest`, existing middleware tests.

---

## Scope

This PR changes only the API middleware authentication path in the public repo. It does not change token generation, logout revocation, Redis blacklist storage, refresh-token handling, dashboard code, Helm, or enterprise deployment.

## Files

- Modify: `apps/api-server/internal/middleware/auth.go`
- Modify: `apps/api-server/internal/middleware/auth_test.go`

## Tasks

### Task 1: Add regression test for invalid token blacklist lookup avoidance

- [ ] Add a test-only `countingBlacklistStore` in `auth_test.go` that increments on `IsRevoked`.
- [ ] Add `TestJWTAuth_InvalidTokenSkipsBlacklistLookup`.
- [ ] Run:

```bash
cd apps/api-server
go test ./internal/middleware -run TestJWTAuth_InvalidTokenSkipsBlacklistLookup -count=1
```

Expected before implementation: FAIL because `IsRevoked` is called once.

### Task 2: Move blacklist check after JWT validation

- [ ] In `JWTAuth`, call `validator.ValidateToken(tokenStr)` before any `hashToken` / `blacklist.IsRevoked`.
- [ ] Keep existing response behavior:
  - invalid/expired token returns `invalid or expired token`,
  - non-access token returns `invalid or expired token`,
  - revoked access token returns `token has been revoked`.

### Task 3: Validate

- [ ] Run:

```bash
cd apps/api-server
go test ./internal/middleware -run 'TestJWTAuth|TestSecurity_Auth' -count=1
```

- [ ] Run:

```bash
cd apps/api-server
go test ./internal/middleware ./internal/service -count=1
```

- [ ] Run:

```bash
cd .
git diff --check
```

- [ ] Before push, run full local CI:

```bash
cd .
./scripts/local-ci.sh
```

## Risk And Rollback

Risk is low: the middleware still checks the same blacklist for valid access tokens, only after cryptographic validation. Rollback is reverting the PR. If a test reveals refresh tokens now hit blacklist before rejection, keep the type check before blacklist unless the revocation model explicitly needs refresh-token blacklist checks on access routes.
