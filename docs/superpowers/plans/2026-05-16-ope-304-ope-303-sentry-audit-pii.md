# OPE-304/OPE-303 Sentry and Audit PII Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent failed-login email addresses and sensitive request material from being persisted in audit logs or sent to Sentry.

**Architecture:** Keep the fix local to the API server. Sentry request scrubbing should happen before request context is attached to the Sentry scope and again in `BeforeSend` as a defense-in-depth backstop. Failed-login audit entries should keep forensic value without storing plaintext email.

**Tech Stack:** Go 1.25, sentry-go, chi middleware, pgx-backed audit repository tests.

---

### Task 1: Sentry Request Scrubbing

**Files:**
- Modify: `apps/api-server/internal/middleware/sentry.go`
- Modify: `apps/api-server/internal/middleware/sentry_test.go`
- Modify: `apps/api-server/cmd/server/main.go`

- [ ] **Step 1: Write a failing middleware test**

Add a test proving Sentry middleware attaches a sanitized request where `Authorization`, `Cookie`, `X-CSRF-Token`, and sensitive query values are absent.

- [ ] **Step 2: Run the focused middleware test**

Run: `cd apps/api-server && go test ./internal/middleware -run TestSentryMiddleware_ScrubsSensitiveRequestData -count=1`

Expected before implementation: fail because request headers/query are still visible.

- [ ] **Step 3: Implement request scrubbing**

Clone the request before `hub.Scope().SetRequest(...)`, remove sensitive headers/cookies, and redact sensitive query values while preserving safe path/method context.

- [ ] **Step 4: Add `BeforeSend` defense-in-depth**

Set `SendDefaultPII: false` and `BeforeSend` in `cmd/server/main.go`, calling a middleware helper that scrubs any request already attached to the event.

- [ ] **Step 5: Run middleware tests**

Run: `cd apps/api-server && go test ./internal/middleware -count=1`

Expected after implementation: pass.

### Task 2: Failed Login Audit PII

**Files:**
- Modify: `apps/api-server/internal/service/auth_service.go`
- Modify: `apps/api-server/internal/service/auth_service_test.go`

- [ ] **Step 1: Write a failing service test**

Add a test that performs a wrong-password login and asserts the emitted `user.login_failed` audit entry does not include the attempted email address in `Changes`.

- [ ] **Step 2: Run the focused service test**

Run: `cd apps/api-server && go test ./internal/service -run TestAuthService_LoginFailedAuditOmitsPlaintextEmail -count=1`

Expected before implementation: fail because `Changes["email"]` currently contains the login email.

- [ ] **Step 3: Implement the minimal audit payload change**

Replace `Changes: map[string]string{"email": req.Email}` with a non-PII marker such as `{"reason": "invalid_password"}`.

- [ ] **Step 4: Run service tests**

Run: `cd apps/api-server && go test ./internal/service -run 'TestAuthService_LoginFailedAuditOmitsPlaintextEmail|TestAuthService_Login_' -count=1`

Expected after implementation: pass.

### Task 3: Validation and PR

**Files:**
- Review all changed files above.
- Update security context docs only if the change creates a durable documented behavior not already covered.

- [ ] **Step 1: Format**

Run: `cd apps/api-server && gofmt -w cmd/server/main.go internal/middleware/sentry.go internal/middleware/sentry_test.go internal/service/auth_service.go internal/service/auth_service_test.go`

- [ ] **Step 2: Focused tests**

Run:

```bash
cd apps/api-server
go test ./internal/middleware -count=1
go test ./internal/service -run 'TestAuthService_LoginFailedAuditOmitsPlaintextEmail|TestAuthService_Login_' -count=1
```

- [ ] **Step 3: Self-review**

Run:

```bash
git diff --check
git diff --stat
git diff
```

Confirm no plaintext failed-login email remains and Sentry request/event scrubbing covers headers, cookies, and sensitive query keys.

- [ ] **Step 4: Full local CI before push**

Run: `./scripts/local-ci.sh`

- [ ] **Step 5: Commit and PR**

Commit with an `OPE-304/OPE-303:` prefix, push the branch, open a PR, then inspect CI and CodeRabbit comments before merge.
