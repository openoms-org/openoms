# OPE-305 Trusted Real IP Rate Limit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop spoofed `X-Forwarded-For` / `X-Real-IP` headers from bypassing IP-based rate limits while preserving real client IP handling behind explicitly trusted production proxies.

**Architecture:** Replace unconditional chi `RealIP` with an OpenOMS `TrustedRealIP` middleware. The middleware changes `r.RemoteAddr` only when the immediate peer is in configured trusted proxy CIDRs; otherwise forwarded headers are ignored and rate limiting uses the TCP peer address. Public chart defaults stay secure with no trusted proxies, while enterprise values must explicitly opt production/staging into the k3s ingress/cloudflared CIDRs before public release deployment.

**Tech Stack:** Go 1.25, chi/v5 middleware chain, `net/netip` for CIDR parsing, Helm chart config, OpenOMS public + enterprise release flow.

---

## Scope

Fix Linear `OPE-305`: `Rate limiter uzywa RemoteAddr po RealIP bez trusted-proxy guard`.

In scope:

- Public API middleware and config.
- Regression tests proving spoofed forwarded headers do not change the limiter key unless the immediate peer is trusted.
- Helm chart value/env wiring for trusted proxy CIDRs.
- Documentation/security posture update.
- Enterprise production/staging values coordination before merging public code, because public `main` auto-releases to production.

Out of scope:

- Replacing Cloudflare WAF/nginx rate limits.
- Creating new rate-limit dimensions by tenant/user beyond the existing authenticated-user limiter.
- Changing webhook signature behavior.
- Hard-coding Cloudflare public IP ranges in the application.

## File Map

Public repo:

- Modify `apps/api-server/internal/config/config.go`  
  Add `TRUSTED_PROXY_CIDRS` config, parse with `netip.ParsePrefix`, validate at startup.
- Modify `apps/api-server/internal/config/config_test.go`  
  Add tests for empty config, multiple CIDRs, whitespace, and invalid CIDR rejection.
- Create `apps/api-server/internal/middleware/trusted_real_ip.go`  
  New middleware that conditionally rewrites `RemoteAddr` only for trusted immediate peers.
- Create `apps/api-server/internal/middleware/trusted_real_ip_test.go`  
  Unit tests for trusted/untrusted peers, `X-Forwarded-For`, `X-Real-IP`, invalid headers, IPv6, and port preservation behavior.
- Modify `apps/api-server/internal/middleware/ratelimit_test.go`  
  Add integration-style regression test proving changing spoofed `X-Forwarded-For` does not bypass limits from an untrusted peer.
- Modify `apps/api-server/internal/middleware/sentry.go`  
  Update middleware-chain comment from `RealIP` to `TrustedRealIP`.
- Modify `apps/api-server/internal/router/router.go`  
  Replace `chimw.RealIP` with `middleware.TrustedRealIP(deps.TrustedProxyCIDRs)`.
- Modify `apps/api-server/cmd/server/main.go`  
  Parse trusted proxy CIDRs after config validation and pass them into `router.RouterDeps`.
- Modify `.env.example`  
  Document `TRUSTED_PROXY_CIDRS`.
- Modify `deploy/helm/openoms/values.yaml`  
  Add `apiServer.trustedProxyCIDRs: []` default.
- Modify `deploy/helm/openoms/templates/configmap.yaml`  
  Render `TRUSTED_PROXY_CIDRS` as a comma-separated string.
- Modify `docs/system-documentation.md`  
  Update middleware stack from `RealIP` to `TrustedRealIP` and document the trust model.
- Update local ignored `.claude/context/SECURITY_POSTURE.md` if needed for AI session context. This file is ignored and is not part of the PR.

Enterprise repo:

- Modify `enterprise/deploy/helm/values-production.yaml`  
  Set trusted proxy CIDRs after read-only validation of actual k3s pod/ingress source ranges.
- Modify `enterprise/deploy/helm/values-staging.yaml`  
  Same for staging if it uses the same ingress/cloudflared path.

## Design Details

Config API:

```go
type Config struct {
    // ...
    TrustedProxyCIDRs string `env:"TRUSTED_PROXY_CIDRS" envDefault:""`
}

func (c *Config) TrustedProxyPrefixes() ([]netip.Prefix, error) {
    if strings.TrimSpace(c.TrustedProxyCIDRs) == "" {
        return nil, nil
    }

    values := strings.Split(c.TrustedProxyCIDRs, ",")
    prefixes := make([]netip.Prefix, 0, len(values))
    for _, value := range values {
        raw := strings.TrimSpace(value)
        if raw == "" {
            continue
        }
        prefix, err := netip.ParsePrefix(raw)
        if err != nil {
            return nil, fmt.Errorf("TRUSTED_PROXY_CIDRS contains invalid CIDR %q: %w", raw, err)
        }
        prefixes = append(prefixes, prefix.Masked())
    }
    return prefixes, nil
}
```

Middleware API:

```go
func TrustedRealIP(trustedProxies []netip.Prefix) func(http.Handler) http.Handler
```

Rules:

- Empty trusted proxy list means no-op: forwarded headers are ignored.
- If `RemoteAddr` cannot be parsed as an IP or `host:port`, no-op.
- If immediate peer IP is not inside any trusted CIDR, no-op.
- If immediate peer is trusted, prefer the first valid IP in `X-Forwarded-For`.
- If `X-Forwarded-For` is missing or has no valid IP, use valid `X-Real-IP`.
- Set `r.RemoteAddr` to the selected client IP without a port, matching chi `RealIP` behavior.
- Do not log forwarded header values.

## Task 1: Branch And Baseline

**Files:**

- No file changes in this task.

- [ ] **Step 1: Confirm clean repos**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
git status --short --branch
cd /Users/rafs/praca/openoms-dev/enterprise
git status --short --branch
```

Expected:

```text
## main...origin/main
```

Enterprise may still show unrelated local artifact `?? chmod`; leave it untouched.

- [ ] **Step 2: Create public branch**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
git checkout -b fix/OPE-305-trusted-real-ip-rate-limit
```

Expected:

```text
Switched to a new branch 'fix/OPE-305-trusted-real-ip-rate-limit'
```

## Task 2: Config Parsing Tests

**Files:**

- Modify `apps/api-server/internal/config/config_test.go`
- Modify `apps/api-server/internal/config/config.go`

- [ ] **Step 1: Write failing config tests**

Append to `apps/api-server/internal/config/config_test.go`:

```go
func TestConfig_TrustedProxyPrefixes(t *testing.T) {
    tests := []struct {
        name    string
        raw     string
        want    []string
        wantErr string
    }{
        {
            name: "empty disables trusted proxies",
            raw:  "",
            want: nil,
        },
        {
            name: "parses comma separated cidrs",
            raw:  "10.42.0.0/16, 127.0.0.1/32, fd00::/8",
            want: []string{"10.42.0.0/16", "127.0.0.1/32", "fd00::/8"},
        },
        {
            name: "trims whitespace and ignores empty segments",
            raw:  " 10.42.0.0/16, , 10.43.0.0/16 ",
            want: []string{"10.42.0.0/16", "10.43.0.0/16"},
        },
        {
            name:    "rejects plain ip without cidr",
            raw:     "10.42.0.10",
            wantErr: "TRUSTED_PROXY_CIDRS contains invalid CIDR",
        },
        {
            name:    "rejects invalid cidr",
            raw:     "not-a-cidr",
            wantErr: "TRUSTED_PROXY_CIDRS contains invalid CIDR",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cfg := Config{TrustedProxyCIDRs: tt.raw}

            got, err := cfg.TrustedProxyPrefixes()
            if tt.wantErr != "" {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.wantErr)
                return
            }

            require.NoError(t, err)
            gotStrings := make([]string, 0, len(got))
            for _, prefix := range got {
                gotStrings = append(gotStrings, prefix.String())
            }
            assert.Equal(t, tt.want, gotStrings)
        })
    }
}

func TestConfig_Validate_RejectsInvalidTrustedProxyCIDRs(t *testing.T) {
    cfg := validConfigForValidation("invite")
    cfg.TrustedProxyCIDRs = "10.42.0.0/16,not-a-cidr"

    err := cfg.Validate()

    require.Error(t, err)
    assert.Contains(t, err.Error(), "TRUSTED_PROXY_CIDRS contains invalid CIDR")
}
```

Also add `net/netip` only after implementation if the compiler requires it for expected types. The test above avoids that import by comparing string values.

- [ ] **Step 2: Run config tests and verify RED**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/config -run 'TestConfig_(TrustedProxyPrefixes|Validate_RejectsInvalidTrustedProxyCIDRs)' -count=1
```

Expected:

```text
FAIL
... cfg.TrustedProxyPrefixes undefined ...
```

- [ ] **Step 3: Implement config parsing**

Modify imports in `apps/api-server/internal/config/config.go`:

```go
import (
    "crypto/ed25519"
    "encoding/base64"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "log/slog"
    "net/netip"
    "strings"

    "github.com/caarlos0/env/v11"
)
```

Add field to `Config` near the existing base URL/frontend URL fields:

```go
// TrustedProxyCIDRs is a comma-separated list of immediate proxy CIDRs whose
// X-Forwarded-For / X-Real-IP headers may update r.RemoteAddr.
TrustedProxyCIDRs string `env:"TRUSTED_PROXY_CIDRS" envDefault:""`
```

Add method near `RequiresRedis()`:

```go
// TrustedProxyPrefixes parses TRUSTED_PROXY_CIDRS into normalized prefixes.
func (c *Config) TrustedProxyPrefixes() ([]netip.Prefix, error) {
    raw := strings.TrimSpace(c.TrustedProxyCIDRs)
    if raw == "" {
        return nil, nil
    }

    parts := strings.Split(raw, ",")
    prefixes := make([]netip.Prefix, 0, len(parts))
    for _, part := range parts {
        value := strings.TrimSpace(part)
        if value == "" {
            continue
        }
        prefix, err := netip.ParsePrefix(value)
        if err != nil {
            return nil, fmt.Errorf("TRUSTED_PROXY_CIDRS contains invalid CIDR %q: %w", value, err)
        }
        prefixes = append(prefixes, prefix.Masked())
    }
    return prefixes, nil
}
```

Add validation in `Validate()` before worker DB validation:

```go
if _, err := c.TrustedProxyPrefixes(); err != nil {
    return err
}
```

- [ ] **Step 4: Run config tests and verify GREEN**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/config -run 'TestConfig_(TrustedProxyPrefixes|Validate_RejectsInvalidTrustedProxyCIDRs)' -count=1
```

Expected:

```text
ok   github.com/openoms-org/openoms/apps/api-server/internal/config
```

## Task 3: TrustedRealIP Middleware Tests

**Files:**

- Create `apps/api-server/internal/middleware/trusted_real_ip_test.go`
- Create `apps/api-server/internal/middleware/trusted_real_ip.go`

- [ ] **Step 1: Write failing middleware tests**

Create `apps/api-server/internal/middleware/trusted_real_ip_test.go`:

```go
package middleware_test

import (
    "net/http"
    "net/http/httptest"
    "net/netip"
    "testing"

    "github.com/openoms-org/openoms/apps/api-server/internal/middleware"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func mustPrefix(t *testing.T, raw string) netip.Prefix {
    t.Helper()
    prefix, err := netip.ParsePrefix(raw)
    require.NoError(t, err)
    return prefix
}

func captureRemoteAddr(t *testing.T, trusted []netip.Prefix, remoteAddr string, headers map[string]string) string {
    t.Helper()
    var got string
    handler := middleware.TrustedRealIP(trusted)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
        got = r.RemoteAddr
    }))

    req := httptest.NewRequest(http.MethodGet, "/", nil)
    req.RemoteAddr = remoteAddr
    for key, value := range headers {
        req.Header.Set(key, value)
    }

    handler.ServeHTTP(httptest.NewRecorder(), req)
    return got
}

func TestTrustedRealIP_IgnoresForwardedHeadersWhenNoTrustedProxiesConfigured(t *testing.T) {
    got := captureRemoteAddr(t, nil, "203.0.113.10:12345", map[string]string{
        "X-Forwarded-For": "198.51.100.20",
        "X-Real-IP":       "198.51.100.21",
    })

    assert.Equal(t, "203.0.113.10:12345", got)
}

func TestTrustedRealIP_IgnoresForwardedHeadersFromUntrustedPeer(t *testing.T) {
    trusted := []netip.Prefix{mustPrefix(t, "10.42.0.0/16")}

    got := captureRemoteAddr(t, trusted, "203.0.113.10:12345", map[string]string{
        "X-Forwarded-For": "198.51.100.20",
        "X-Real-IP":       "198.51.100.21",
    })

    assert.Equal(t, "203.0.113.10:12345", got)
}

func TestTrustedRealIP_UsesXForwardedForFromTrustedPeer(t *testing.T) {
    trusted := []netip.Prefix{mustPrefix(t, "10.42.0.0/16")}

    got := captureRemoteAddr(t, trusted, "10.42.0.25:5678", map[string]string{
        "X-Forwarded-For": "198.51.100.20, 10.42.0.25",
        "X-Real-IP":       "198.51.100.21",
    })

    assert.Equal(t, "198.51.100.20", got)
}

func TestTrustedRealIP_UsesXRealIPFallbackFromTrustedPeer(t *testing.T) {
    trusted := []netip.Prefix{mustPrefix(t, "10.42.0.0/16")}

    got := captureRemoteAddr(t, trusted, "10.42.0.25:5678", map[string]string{
        "X-Real-IP": "198.51.100.21",
    })

    assert.Equal(t, "198.51.100.21", got)
}

func TestTrustedRealIP_IgnoresInvalidForwardedHeaders(t *testing.T) {
    trusted := []netip.Prefix{mustPrefix(t, "10.42.0.0/16")}

    got := captureRemoteAddr(t, trusted, "10.42.0.25:5678", map[string]string{
        "X-Forwarded-For": "not-an-ip",
        "X-Real-IP":       "also-not-an-ip",
    })

    assert.Equal(t, "10.42.0.25:5678", got)
}

func TestTrustedRealIP_SupportsIPv6TrustedPeer(t *testing.T) {
    trusted := []netip.Prefix{mustPrefix(t, "fd00::/8")}

    got := captureRemoteAddr(t, trusted, "[fd00::10]:5678", map[string]string{
        "X-Forwarded-For": "2001:db8::5",
    })

    assert.Equal(t, "2001:db8::5", got)
}
```

- [ ] **Step 2: Run middleware tests and verify RED**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/middleware -run TestTrustedRealIP -count=1
```

Expected:

```text
FAIL
... undefined: middleware.TrustedRealIP ...
```

- [ ] **Step 3: Implement middleware**

Create `apps/api-server/internal/middleware/trusted_real_ip.go`:

```go
package middleware

import (
    "net"
    "net/http"
    "net/netip"
    "strings"
)

// TrustedRealIP updates r.RemoteAddr from forwarding headers only when the
// immediate peer is an explicitly trusted proxy. With no trusted proxies it is
// a no-op, so spoofed X-Forwarded-For headers cannot affect rate-limit keys.
func TrustedRealIP(trustedProxies []netip.Prefix) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if len(trustedProxies) == 0 {
                next.ServeHTTP(w, r)
                return
            }

            peerIP, ok := remoteAddrIP(r.RemoteAddr)
            if !ok || !isTrustedProxy(peerIP, trustedProxies) {
                next.ServeHTTP(w, r)
                return
            }

            if clientIP, ok := forwardedClientIP(r.Header.Get("X-Forwarded-For")); ok {
                r.RemoteAddr = clientIP.String()
                next.ServeHTTP(w, r)
                return
            }

            if clientIP, ok := headerIP(r.Header.Get("X-Real-IP")); ok {
                r.RemoteAddr = clientIP.String()
            }

            next.ServeHTTP(w, r)
        })
    }
}

func remoteAddrIP(remoteAddr string) (netip.Addr, bool) {
    host, _, err := net.SplitHostPort(remoteAddr)
    if err != nil {
        host = remoteAddr
    }
    return headerIP(host)
}

func forwardedClientIP(value string) (netip.Addr, bool) {
    for _, part := range strings.Split(value, ",") {
        if ip, ok := headerIP(part); ok {
            return ip, true
        }
    }
    return netip.Addr{}, false
}

func headerIP(value string) (netip.Addr, bool) {
    raw := strings.TrimSpace(value)
    if raw == "" {
        return netip.Addr{}, false
    }
    ip, err := netip.ParseAddr(raw)
    if err != nil {
        return netip.Addr{}, false
    }
    return ip.Unmap(), true
}

func isTrustedProxy(ip netip.Addr, trustedProxies []netip.Prefix) bool {
    ip = ip.Unmap()
    for _, trusted := range trustedProxies {
        if trusted.Contains(ip) {
            return true
        }
    }
    return false
}
```

- [ ] **Step 4: Run middleware tests and verify GREEN**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/middleware -run TestTrustedRealIP -count=1
```

Expected:

```text
ok   github.com/openoms-org/openoms/apps/api-server/internal/middleware
```

## Task 4: Rate Limit Regression Test

**Files:**

- Modify `apps/api-server/internal/middleware/ratelimit_test.go`

- [ ] **Step 1: Write failing regression test**

Add imports to `ratelimit_test.go`:

```go
import (
    // existing imports
    "net/netip"
)
```

Append test near `TestRateLimitWith_ExceededReturns429`:

```go
func TestRateLimitWith_ChangingSpoofedXForwardedForDoesNotBypassUntrustedPeer(t *testing.T) {
    trusted := []netip.Prefix{netip.MustParsePrefix("10.42.0.0/16")}
    limiter := middleware.NewMemoryRateLimiter()
    handler := middleware.TrustedRealIP(trusted)(
        middleware.RateLimitWith(limiter, 2, time.Minute)(testOKHandler()),
    )

    spoofedIPs := []string{"198.51.100.10", "198.51.100.11", "198.51.100.12"}
    for i, spoofedIP := range spoofedIPs {
        req := httptest.NewRequest("POST", "/v1/auth/login", nil)
        req.RemoteAddr = "203.0.113.50:12345"
        req.Header.Set("X-Forwarded-For", spoofedIP)
        rr := httptest.NewRecorder()

        handler.ServeHTTP(rr, req)

        if i < 2 {
            assert.Equal(t, http.StatusOK, rr.Code, "request %d should be allowed", i+1)
            continue
        }
        assert.Equal(t, http.StatusTooManyRequests, rr.Code, "spoofed X-Forwarded-For must not reset the limiter key")
    }
}
```

This test compiles only after Task 3. Its security value is the behavior: different spoofed headers from the same untrusted peer still hit the same limiter key.

- [ ] **Step 2: Run targeted regression**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/middleware -run 'Test(RateLimitWith_ChangingSpoofedXForwardedForDoesNotBypassUntrustedPeer|TrustedRealIP)' -count=1
```

Expected:

```text
ok   github.com/openoms-org/openoms/apps/api-server/internal/middleware
```

## Task 5: Wire Middleware Into Startup And Router

**Files:**

- Modify `apps/api-server/internal/router/router.go`
- Modify `apps/api-server/cmd/server/main.go`

- [ ] **Step 1: Add trusted proxy prefixes to router deps**

In `apps/api-server/internal/router/router.go`, add import:

```go
import (
    "net/netip"
    // existing imports
)
```

Add field to `RouterDeps`:

```go
TrustedProxyCIDRs []netip.Prefix
```

Replace:

```go
r.Use(chimw.RealIP)
```

with:

```go
r.Use(middleware.TrustedRealIP(deps.TrustedProxyCIDRs))
```

Keep `chimw` import because `RequestID` and `Recoverer` still use chi middleware.

- [ ] **Step 2: Parse config before router creation**

In `apps/api-server/cmd/server/main.go`, after `cfg.Validate()` has passed and before `router.New(...)`, add:

```go
trustedProxyCIDRs, err := cfg.TrustedProxyPrefixes()
if err != nil {
    slog.Error("failed to parse TRUSTED_PROXY_CIDRS", "error", err)
    return fmt.Errorf("failed to parse TRUSTED_PROXY_CIDRS: %w", err)
}
```

In the `router.RouterDeps{...}` literal, add:

```go
TrustedProxyCIDRs: trustedProxyCIDRs,
```

- [ ] **Step 3: Run backend package tests touched by wiring**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/config ./internal/middleware ./internal/router -count=1
```

Expected:

```text
ok   github.com/openoms-org/openoms/apps/api-server/internal/config
ok   github.com/openoms-org/openoms/apps/api-server/internal/middleware
ok   github.com/openoms-org/openoms/apps/api-server/internal/router
```

If `internal/router` has no tests or requires unavailable dependencies, run:

```bash
go test ./internal/router -run '^$' -count=1
```

Expected: package compiles successfully.

## Task 6: Helm And Env Wiring

**Files:**

- Modify `.env.example`
- Modify `deploy/helm/openoms/values.yaml`
- Modify `deploy/helm/openoms/templates/configmap.yaml`

- [ ] **Step 1: Add env example**

Add under `Auth / Security` in `.env.example`:

```dotenv
# Comma-separated immediate proxy CIDRs allowed to provide X-Forwarded-For/X-Real-IP.
# Empty means forwarded headers are ignored. Example for k3s ingress pods: 10.42.0.0/16
TRUSTED_PROXY_CIDRS=
```

- [ ] **Step 2: Add Helm default**

Add under `apiServer:` in `deploy/helm/openoms/values.yaml` near `frontendUrl`:

```yaml
  # Comma-rendered to TRUSTED_PROXY_CIDRS. Empty means forwarded headers are ignored.
  # Set only to CIDRs of immediate trusted proxies, such as ingress/cloudflared pod CIDRs.
  trustedProxyCIDRs: []
```

- [ ] **Step 3: Render ConfigMap env var**

Add to `deploy/helm/openoms/templates/configmap.yaml` near `FRONTEND_URL`:

```yaml
  TRUSTED_PROXY_CIDRS: {{ join "," .Values.apiServer.trustedProxyCIDRs | quote }}
```

- [ ] **Step 4: Validate Helm default and override rendering**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
helm template openoms deploy/helm/openoms >/tmp/openoms-ope-305-default.yaml
helm template openoms deploy/helm/openoms --set 'apiServer.trustedProxyCIDRs[0]=10.42.0.0/16' >/tmp/openoms-ope-305-trusted.yaml
rg -n 'TRUSTED_PROXY_CIDRS' /tmp/openoms-ope-305-default.yaml /tmp/openoms-ope-305-trusted.yaml
```

Expected:

```text
/tmp/openoms-ope-305-default.yaml:...:  TRUSTED_PROXY_CIDRS: ""
/tmp/openoms-ope-305-trusted.yaml:...:  TRUSTED_PROXY_CIDRS: "10.42.0.0/16"
```

## Task 7: Documentation Updates

**Files:**

- Modify `docs/system-documentation.md`
- Modify `.claude/context/SECURITY_POSTURE.md`

- [ ] **Step 1: Update system documentation middleware stack**

Change:

```text
Request -> RequestID -> RealIP -> Prometheus -> SecurityHeaders -> CSRF -> HSTS -> Logger -> Recoverer -> CORS
```

to:

```text
Request -> RequestID -> TrustedRealIP -> Prometheus -> SecurityHeaders -> CSRF -> HSTS -> Logger -> Recoverer -> CORS
```

Add a short paragraph below the middleware stack:

```md
`TrustedRealIP` honors `X-Forwarded-For` / `X-Real-IP` only when the immediate peer IP is inside `TRUSTED_PROXY_CIDRS`. With an empty list, forwarded headers are ignored and rate limits/logs use the TCP peer address. Production values must set only the CIDRs of ingress/cloudflared pods that sanitize or control forwarded headers.
```

- [ ] **Step 2: Update security posture**

Add under `Security updates (2026-05-05 to 2026-05-16)`:

```md
- OPE-305: chi `RealIP` was replaced with `TrustedRealIP`, so `X-Forwarded-For` and `X-Real-IP` affect `RemoteAddr` only when the immediate peer is in `TRUSTED_PROXY_CIDRS`. Empty config ignores forwarded headers, preventing spoofed header rotation from bypassing IP-based login/register rate limits.
```

## Task 8: Enterprise Values Coordination

**Files:**

- Modify `/Users/rafs/praca/openoms-dev/enterprise/deploy/helm/values-production.yaml`
- Modify `/Users/rafs/praca/openoms-dev/enterprise/deploy/helm/values-staging.yaml`

This task is required before merging the public PR, because public `main` triggers a release/deploy.

- [ ] **Step 1: Read-only verify proxy source ranges**

Run only read-only commands:

```bash
export KUBECONFIG=~/.kube/openoms-config
kubectl --context openoms-vpn get nodes -o jsonpath='{range .items[*]}{.metadata.name}{" podCIDR="}{.spec.podCIDR}{" podCIDRs="}{.spec.podCIDRs}{"\n"}{end}'
kubectl --context openoms-vpn -n ingress-nginx get pods -o wide
kubectl --context openoms-vpn -n cloudflared get pods -o wide
```

Expected:

- Node pod CIDRs show the k3s pod ranges used by ingress/cloudflared pods.
- ingress-nginx and cloudflared pod IPs fall inside the chosen CIDR.

- [ ] **Step 2: Add enterprise values**

For the 2026-05-16 cluster snapshot, read-only validation showed node pod CIDRs `10.42.0.0/24` and `10.42.1.0/24`, with ingress-nginx/cloudflared pods currently in `10.42.0.0/24`. Add the verified node pod CIDRs to both production and staging overlays under `apiServer:`:

```yaml
  trustedProxyCIDRs:
    - 10.42.0.0/24
    - 10.42.1.0/24
```

If validation shows different immediate proxy CIDRs in a future cluster, use the verified node pod CIDRs instead of a broader cluster-wide range.

- [ ] **Step 3: Validate enterprise Helm rendering**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/enterprise
helm template openoms ../public/deploy/helm/openoms -f deploy/helm/values-production.yaml >/tmp/openoms-ope-305-prod.yaml
helm template openoms ../public/deploy/helm/openoms -f deploy/helm/values-staging.yaml >/tmp/openoms-ope-305-staging.yaml
rg -n 'TRUSTED_PROXY_CIDRS' /tmp/openoms-ope-305-prod.yaml /tmp/openoms-ope-305-staging.yaml
```

Expected:

```text
TRUSTED_PROXY_CIDRS: "10.42.0.0/24,10.42.1.0/24"
```

## Task 9: Self-Review And Validation

**Files:**

- All changed files.

- [ ] **Step 1: Format Go**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
gofmt -w -s apps/api-server/internal/config/config.go apps/api-server/internal/config/config_test.go apps/api-server/internal/middleware/trusted_real_ip.go apps/api-server/internal/middleware/trusted_real_ip_test.go apps/api-server/internal/middleware/ratelimit_test.go apps/api-server/internal/router/router.go apps/api-server/cmd/server/main.go
```

Expected: no output.

- [ ] **Step 2: Targeted tests**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/config ./internal/middleware ./internal/router -count=1
```

Expected: all packages pass.

- [ ] **Step 3: Helm checks**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
helm template openoms deploy/helm/openoms >/tmp/openoms-ope-305-default.yaml
helm template openoms deploy/helm/openoms --set 'apiServer.trustedProxyCIDRs[0]=10.42.0.0/16' >/tmp/openoms-ope-305-trusted.yaml
```

Expected: both commands exit 0.

- [ ] **Step 4: Diff self-review**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
git diff --check
git diff --stat
git diff
```

Check:

- No unrelated files.
- No secrets or live IP/token values.
- `chimw.RealIP` is gone from runtime middleware chain.
- Tests demonstrate trusted and untrusted proxy behavior.
- Docs reflect the new trust model.

- [ ] **Step 5: Full public local CI before push**

Run after commit on clean branch:

```bash
cd /Users/rafs/praca/openoms-dev/public
./scripts/local-ci.sh
```

Expected:

```text
STATUS=pass
```

## Task 10: Commit, PR, Review, Merge Order

**Files:**

- Git metadata only after validation.

- [ ] **Step 1: Commit public changes**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
git add apps/api-server/internal/config/config.go apps/api-server/internal/config/config_test.go apps/api-server/internal/middleware/trusted_real_ip.go apps/api-server/internal/middleware/trusted_real_ip_test.go apps/api-server/internal/middleware/ratelimit_test.go apps/api-server/internal/middleware/sentry.go apps/api-server/internal/router/router.go apps/api-server/cmd/server/main.go .env.example deploy/helm/openoms/values.yaml deploy/helm/openoms/templates/configmap.yaml docs/system-documentation.md docs/superpowers/plans/2026-05-16-ope-305-trusted-real-ip-rate-limit.md
git commit -m "OPE-305: add trusted real ip handling"
```

- [ ] **Step 2: Commit enterprise values if needed**

Use a separate enterprise branch and PR if values changed:

```bash
cd /Users/rafs/praca/openoms-dev/enterprise
git checkout -b fix/OPE-305-trusted-proxy-values
git add deploy/helm/values-production.yaml deploy/helm/values-staging.yaml
git commit -m "OPE-305: configure trusted proxy cidrs"
```

- [ ] **Step 3: Merge order**

1. Open and merge enterprise values PR first if public default would otherwise degrade production real-client IP behavior.
2. Open public PR `OPE-305: add trusted real IP handling`.
3. Inspect CI, CodeQL, and CodeRabbit comments/review threads.
4. Merge public PR only after checks pass and CodeRabbit has no unresolved actionable comments.
5. Validate the release/deploy and production smoke after public merge.

## Risk And Rollback

Risk: If `TRUSTED_PROXY_CIDRS` is empty in production behind ingress, all IP-based unauthenticated limits use the ingress/cloudflared peer IP. This is secure against spoofing but can over-limit legitimate users globally.

Mitigation: Merge enterprise values before public release deploy, using verified k3s pod CIDRs.

Risk: If `TRUSTED_PROXY_CIDRS` is too broad, a direct client from a trusted range can spoof forwarded headers.

Mitigation: Configure only immediate proxy pod CIDRs or a narrow private ingress range, never `0.0.0.0/0` or broad office/VPN ranges. Config validation rejects malformed CIDRs but cannot know operational trust boundaries.

Rollback:

- Public rollback: revert the public PR; chi `RealIP` behavior returns only if explicitly restored.
- Safer operational rollback: keep public code and adjust enterprise `apiServer.trustedProxyCIDRs` to the verified immediate proxy CIDR, then redeploy.
- Emergency rate-limit rollback: temporarily increase login/register edge WAF limits or ingress limits while fixing CIDR values; do not set `TRUSTED_PROXY_CIDRS=0.0.0.0/0`.

## Final Validation Checklist

- [ ] `go test ./internal/config ./internal/middleware ./internal/router -count=1` passes.
- [ ] `helm template` default and trusted-proxy override both render.
- [ ] `git diff --check` passes.
- [ ] Full `./scripts/local-ci.sh` passes before push.
- [ ] Enterprise values are merged or explicitly not needed before public merge.
- [ ] PR body includes docs updated section.
- [ ] CodeRabbit comments/review threads are read and resolved or consciously deferred.
- [ ] Production deploy validates `/health`, dashboard login page, and no fresh API restart delta.

## CodeRabbit Follow-Up

2026-05-16 review found one valid issue: left-to-right `X-Forwarded-For` parsing can select a client-supplied spoofed leftmost value after a trusted proxy appends the real peer IP. Fix with TDD:

- Add regression for trusted peer with `X-Forwarded-For: 203.0.113.200, 198.51.100.20, 10.42.0.25`; expected resolved client is `198.51.100.20`.
- Update `forwardedClientIP` to parse valid candidates and scan right-to-left, returning the first address outside `TRUSTED_PROXY_CIDRS`.
- Keep `X-Real-IP` as fallback when no safe XFF candidate exists.
