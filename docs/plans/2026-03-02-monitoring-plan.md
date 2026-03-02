# Sentry + Grafana Cloud Monitoring — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add production-grade error tracking (Sentry) and metrics dashboards (Grafana Cloud) to the Go API server and Next.js dashboard.

**Architecture:** Sentry SaaS captures errors/panics from both Go API (via `sentry-go` middleware + worker hooks) and Next.js dashboard (via `@sentry/nextjs`). Existing Prometheus `/metrics` endpoint is scraped by Grafana Alloy agent on k3s and forwarded to Grafana Cloud for dashboards and alerts.

**Tech Stack:** `sentry-go` v0.31+, `@sentry/nextjs` v9+, Grafana Alloy, Helm

---

### Task 1: Go — Add sentry-go dependency + config

**Files:**
- Modify: `apps/api-server/go.mod`
- Modify: `apps/api-server/internal/config/config.go`

**Step 1: Add sentry-go dependency**

```bash
cd apps/api-server && go get github.com/getsentry/sentry-go@latest
```

**Step 2: Add Sentry config fields to Config struct**

In `apps/api-server/internal/config/config.go`, add after the `BillingPlansJSON` field (line 67):

```go
	// Sentry error tracking. Empty DSN = Sentry disabled (self-hosted mode).
	SentryDSN             string  `env:"SENTRY_DSN" envDefault:""`
	SentryEnvironment     string  `env:"SENTRY_ENVIRONMENT" envDefault:""`
	SentryTracesSampleRate float64 `env:"SENTRY_TRACES_SAMPLE_RATE" envDefault:"0"`
```

**Step 3: Add SentryEnabled helper**

After `BillingEnabled()` method (line 117), add:

```go
// SentryEnabled returns true when Sentry error tracking is configured.
func (c *Config) SentryEnabled() bool {
	return c.SentryDSN != ""
}

// SentryEnv returns the Sentry environment, defaulting to ENV value.
func (c *Config) SentryEnv() string {
	if c.SentryEnvironment != "" {
		return c.SentryEnvironment
	}
	return c.Env
}
```

**Step 4: Verify it compiles**

```bash
cd apps/api-server && go build ./...
```

**Step 5: Commit**

```bash
git add apps/api-server/go.mod apps/api-server/go.sum apps/api-server/internal/config/config.go
git commit -m "feat: add sentry-go dependency and config fields"
```

---

### Task 2: Go — Sentry init in main.go + graceful shutdown

**Files:**
- Modify: `apps/api-server/cmd/server/main.go`

**Step 1: Add Sentry import and init**

In `cmd/server/main.go`, add to imports:

```go
	"github.com/getsentry/sentry-go"
```

After the logger setup (after line 102 `slog.SetDefault(slog.New(logHandler))`), add Sentry init:

```go
	// Initialize Sentry error tracking (optional — disabled when DSN is empty)
	if cfg.SentryEnabled() {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDSN,
			Environment:      cfg.SentryEnv(),
			Release:          version,
			TracesSampleRate: cfg.SentryTracesSampleRate,
			EnableTracing:    cfg.SentryTracesSampleRate > 0,
		})
		if err != nil {
			slog.Error("failed to initialize Sentry", "error", err)
		} else {
			slog.Info("Sentry initialized", "environment", cfg.SentryEnv())
		}
		defer sentry.Flush(2 * time.Second)
	}
```

Note: If `version` variable does not exist, define it at package level:

```go
var version = "dev" // overridden by -ldflags at build time
```

**Step 2: Verify it compiles**

```bash
cd apps/api-server && go build ./...
```

**Step 3: Commit**

```bash
git add apps/api-server/cmd/server/main.go
git commit -m "feat: initialize Sentry in main with graceful flush"
```

---

### Task 3: Go — Sentry middleware for HTTP requests

**Files:**
- Create: `apps/api-server/internal/middleware/sentry.go`
- Modify: `apps/api-server/internal/router/router.go`

**Step 1: Create Sentry middleware**

Create `apps/api-server/internal/middleware/sentry.go`:

```go
package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/getsentry/sentry-go"
)

// SentryMiddleware captures panics and reports them to Sentry with request context.
// Should be placed early in the middleware chain (after RealIP, before Recoverer).
func SentryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.GetHubFromContext(r.Context())
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
		}
		hub.Scope().SetRequest(r)

		ctx := sentry.SetHubOnContext(r.Context(), hub)

		defer func() {
			if err := recover(); err != nil {
				hub.RecoverWithContext(ctx, err)
				sentry.Flush(2 * time.Second)

				// Re-panic so chi's Recoverer middleware can handle the HTTP response.
				panic(err)
			}
		}()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

Wait — we need the `time` import. Update the file:

```go
package middleware

import (
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
)

// SentryMiddleware captures panics and reports them to Sentry with request context.
// Should be placed early in the middleware chain (after RealIP, before Recoverer).
func SentryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.GetHubFromContext(r.Context())
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
		}
		hub.Scope().SetRequest(r)

		ctx := sentry.SetHubOnContext(r.Context(), hub)

		defer func() {
			if err := recover(); err != nil {
				hub.RecoverWithContext(ctx, err)
				sentry.Flush(2 * time.Second)
				// Re-panic so chi's Recoverer handles the HTTP 500 response.
				panic(err)
			}
		}()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

**Step 2: Wire middleware into router**

In `apps/api-server/internal/router/router.go`, add the Sentry middleware after `chimw.RealIP` (line 119) and before `MetricsCollector` (line 120). The middleware should be conditional (only if Sentry is initialized):

After line 119 (`r.Use(chimw.RealIP)`), add:

```go
	if sentry.CurrentHub().Client() != nil {
		r.Use(middleware.SentryMiddleware)
	}
```

Add `"github.com/getsentry/sentry-go"` to imports.

**Step 3: Verify it compiles**

```bash
cd apps/api-server && go build ./...
```

**Step 4: Commit**

```bash
git add apps/api-server/internal/middleware/sentry.go apps/api-server/internal/router/router.go
git commit -m "feat: add Sentry middleware for HTTP panic capture"
```

---

### Task 4: Go — Sentry capture in worker panic recovery + SafeGo

**Files:**
- Modify: `apps/api-server/internal/worker/manager.go`
- Modify: `apps/api-server/internal/asyncutil/safego.go`

**Step 1: Add Sentry capture to worker safeRun**

In `apps/api-server/internal/worker/manager.go`, modify `safeRun()` (lines 96-105):

Replace the existing `safeRun` method with:

```go
func (m *Manager) safeRun(ctx context.Context, w Worker) {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			slog.Error("worker panicked", "worker", w.Name(), "error", r, "stack", stack)
			if hub := sentry.GetHubFromContext(ctx); hub != nil {
				hub.WithScope(func(scope *sentry.Scope) {
					scope.SetTag("worker", w.Name())
					hub.RecoverWithContext(ctx, r)
				})
			} else {
				sentry.CurrentHub().WithScope(func(scope *sentry.Scope) {
					scope.SetTag("worker", w.Name())
					sentry.CurrentHub().RecoverWithContext(ctx, r)
				})
			}
		}
	}()
	if err := w.Run(ctx); err != nil {
		m.logger.Error("worker run failed", "name", w.Name(), "error", err)
		sentry.WithScope(func(scope *sentry.Scope) {
			scope.SetTag("worker", w.Name())
			scope.SetLevel(sentry.LevelError)
			sentry.CaptureException(err)
		})
	}
}
```

Add `"github.com/getsentry/sentry-go"` to imports.

**Step 2: Add Sentry capture to SafeGo**

In `apps/api-server/internal/asyncutil/safego.go`, replace the entire file:

```go
// Package asyncutil provides helpers for safe asynchronous execution.
package asyncutil

import (
	"log/slog"
	"runtime/debug"

	"github.com/getsentry/sentry-go"
)

// SafeGo runs fn in a new goroutine with panic recovery.
// If fn panics, the panic is logged, reported to Sentry, and the goroutine exits cleanly.
func SafeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("goroutine panicked", "error", r, "stack", string(debug.Stack()))
				sentry.CurrentHub().Recover(r)
			}
		}()
		fn()
	}()
}
```

**Step 3: Verify it compiles**

```bash
cd apps/api-server && go build ./...
```

**Step 4: Commit**

```bash
git add apps/api-server/internal/worker/manager.go apps/api-server/internal/asyncutil/safego.go
git commit -m "feat: add Sentry capture to worker panic recovery and SafeGo"
```

---

### Task 5: Go — Sentry context enrichment (tenant_id, user_id)

**Files:**
- Create: `apps/api-server/internal/middleware/sentry_context.go`
- Modify: `apps/api-server/internal/router/router.go`

**Step 1: Create Sentry context middleware**

This middleware runs AFTER JWT auth and adds tenant_id + user_id to Sentry scope. It goes on authenticated routes.

Create `apps/api-server/internal/middleware/sentry_context.go`:

```go
package middleware

import (
	"net/http"

	"github.com/getsentry/sentry-go"
)

// SentryContext enriches Sentry events with tenant and user context.
// Must be placed after JWTAuth middleware (needs claims in context).
func SentryContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.GetHubFromContext(r.Context())
		if hub == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Extract claims from context (set by JWTAuth middleware)
		if tenantID, ok := r.Context().Value("tenant_id").(string); ok && tenantID != "" {
			hub.Scope().SetTag("tenant_id", tenantID)
		}
		if userID, ok := r.Context().Value("user_id").(string); ok && userID != "" {
			hub.Scope().SetUser(sentry.User{ID: userID})
		}

		next.ServeHTTP(w, r)
	})
}
```

Note: Check how JWT claims are stored in context. Look at `middleware/auth.go` to find the exact context key names used. The keys might be typed (e.g., `type contextKey string`) — match them exactly.

**Step 2: Wire into authenticated routes**

In `router.go`, find where the authenticated route group is defined (after JWT middleware is applied) and add `middleware.SentryContext`. It should be used right after `RequireAuth` or `JWTAuth` in the protected route group.

**Step 3: Verify it compiles**

```bash
cd apps/api-server && go build ./...
```

**Step 4: Commit**

```bash
git add apps/api-server/internal/middleware/sentry_context.go apps/api-server/internal/router/router.go
git commit -m "feat: enrich Sentry events with tenant and user context"
```

---

### Task 6: Go — Sentry capture in writeServerError

**Files:**
- Modify: `apps/api-server/internal/handler/response.go`

**Step 1: Add Sentry capture to writeServerError**

In `apps/api-server/internal/handler/response.go`, modify `writeServerError` (line 34) to also capture the error in Sentry:

Replace the existing function:

```go
// writeServerError logs the underlying error, reports it to Sentry, and returns a generic message to the client.
func writeServerError(w http.ResponseWriter, message string, err error) {
	slog.Error(message, "error", err)
	sentry.CaptureException(err)
	writeError(w, http.StatusInternalServerError, message)
}
```

Add `"github.com/getsentry/sentry-go"` to imports.

Note: This captures errors from ALL handlers that call `writeServerError()`, which is the standard pattern for 500 responses in the codebase.

**Step 2: Verify it compiles**

```bash
cd apps/api-server && go build ./...
```

**Step 3: Run tests**

```bash
cd apps/api-server && go test ./internal/handler/... 2>&1 | tail -5
```

**Step 4: Commit**

```bash
git add apps/api-server/internal/handler/response.go
git commit -m "feat: capture 5xx errors in Sentry via writeServerError"
```

---

### Task 7: Go — Tests for Sentry integration

**Files:**
- Create: `apps/api-server/internal/middleware/sentry_test.go`

**Step 1: Write tests**

Create `apps/api-server/internal/middleware/sentry_test.go`:

```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
)

func TestSentryMiddleware_NormalRequest(t *testing.T) {
	// Initialize Sentry in test mode (no actual transport)
	err := sentry.Init(sentry.ClientOptions{
		Dsn: "",
	})
	assert.NoError(t, err)

	handler := middleware.SentryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSentryMiddleware_PanicRecovery(t *testing.T) {
	err := sentry.Init(sentry.ClientOptions{
		Dsn: "",
	})
	assert.NoError(t, err)

	handler := middleware.SentryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// SentryMiddleware re-panics after capturing — expect panic
	assert.Panics(t, func() {
		handler.ServeHTTP(rec, req)
	})
}
```

**Step 2: Run tests**

```bash
cd apps/api-server && go test ./internal/middleware/... 2>&1 | tail -5
```

**Step 3: Commit**

```bash
git add apps/api-server/internal/middleware/sentry_test.go
git commit -m "test: add Sentry middleware tests"
```

---

### Task 8: Next.js — Install @sentry/nextjs + setup wizard files

**Files:**
- Modify: `apps/dashboard/package.json`
- Create: `apps/dashboard/sentry.client.config.ts`
- Create: `apps/dashboard/sentry.server.config.ts`
- Create: `apps/dashboard/sentry.edge.config.ts`
- Create: `apps/dashboard/src/instrumentation.ts`

**Step 1: Install Sentry SDK**

```bash
cd apps/dashboard && npm install @sentry/nextjs
```

**Step 2: Create sentry.client.config.ts**

Create `apps/dashboard/sentry.client.config.ts`:

```typescript
import * as Sentry from "@sentry/nextjs";

Sentry.init({
  dsn: process.env.NEXT_PUBLIC_SENTRY_DSN,
  environment: process.env.NEXT_PUBLIC_SENTRY_ENVIRONMENT || "development",

  // Performance monitoring — low sample rate in production
  tracesSampleRate: process.env.NODE_ENV === "production" ? 0.1 : 1.0,

  // Session replay — disabled by default (adds bundle size)
  replaysSessionSampleRate: 0,
  replaysOnErrorSampleRate: 0,

  // Only enable in production (skip localhost noise)
  enabled: process.env.NODE_ENV === "production",
});
```

**Step 3: Create sentry.server.config.ts**

Create `apps/dashboard/sentry.server.config.ts`:

```typescript
import * as Sentry from "@sentry/nextjs";

Sentry.init({
  dsn: process.env.NEXT_PUBLIC_SENTRY_DSN,
  environment: process.env.NEXT_PUBLIC_SENTRY_ENVIRONMENT || "development",
  tracesSampleRate: process.env.NODE_ENV === "production" ? 0.1 : 1.0,
  enabled: process.env.NODE_ENV === "production",
});
```

**Step 4: Create sentry.edge.config.ts**

Create `apps/dashboard/sentry.edge.config.ts`:

```typescript
import * as Sentry from "@sentry/nextjs";

Sentry.init({
  dsn: process.env.NEXT_PUBLIC_SENTRY_DSN,
  environment: process.env.NEXT_PUBLIC_SENTRY_ENVIRONMENT || "development",
  tracesSampleRate: process.env.NODE_ENV === "production" ? 0.1 : 1.0,
  enabled: process.env.NODE_ENV === "production",
});
```

**Step 5: Create instrumentation.ts**

Create `apps/dashboard/src/instrumentation.ts`:

```typescript
export async function register() {
  if (process.env.NEXT_RUNTIME === "nodejs") {
    await import("../sentry.server.config");
  }

  if (process.env.NEXT_RUNTIME === "edge") {
    await import("../sentry.edge.config");
  }
}

export const onRequestError = async (
  err: { digest: string } & Error,
  request: {
    path: string;
    method: string;
    headers: { [key: string]: string };
  },
  context: {
    routerKind: "Pages Router" | "App Router";
    routePath: string;
    routeType: "render" | "route" | "action" | "middleware";
    renderType: "ssr" | "ssg" | "isr";
  }
) => {
  const { captureRequestError } = await import("@sentry/nextjs");
  captureRequestError(err, request, context);
};
```

**Step 6: Verify it builds**

```bash
cd apps/dashboard && npx next build 2>&1 | tail -10
```

**Step 7: Commit**

```bash
git add apps/dashboard/package.json apps/dashboard/package-lock.json apps/dashboard/sentry.client.config.ts apps/dashboard/sentry.server.config.ts apps/dashboard/sentry.edge.config.ts apps/dashboard/src/instrumentation.ts
git commit -m "feat: add Sentry SDK to Next.js dashboard"
```

---

### Task 9: Next.js — Wrap next.config.ts with Sentry + CSP + global-error

**Files:**
- Modify: `apps/dashboard/next.config.ts`
- Create: `apps/dashboard/src/app/global-error.tsx`

**Step 1: Wrap next.config.ts with withSentryConfig**

Modify `apps/dashboard/next.config.ts` to wrap with Sentry and add `*.sentry.io` to CSP:

```typescript
import type { NextConfig } from "next";
import { withSentryConfig } from "@sentry/nextjs";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

function getWsDirectives(): string {
  try {
    const { hostname } = new URL(apiUrl);
    return `wss://${hostname} ws://${hostname}`;
  } catch {
    return "wss://WS_CSP_HOST_PLACEHOLDER ws://WS_CSP_HOST_PLACEHOLDER";
  }
}

const nextConfig: NextConfig = {
  output: "standalone",
  poweredByHeader: false,
  productionBrowserSourceMaps: false,
  redirects: async () => [
    { source: "/integrations/allegro", destination: "/marketplaces/allegro", permanent: true },
    { source: "/integrations/allegro/:path*", destination: "/marketplaces/allegro/:path*", permanent: true },
    { source: "/integrations/amazon", destination: "/marketplaces/amazon", permanent: true },
    { source: "/integrations/shoper", destination: "/marketplaces/shoper", permanent: true },
    { source: "/integrations/prestashop", destination: "/marketplaces/prestashop", permanent: true },
    { source: "/integrations/shopify", destination: "/marketplaces/shopify", permanent: true },
    { source: "/settings/invoicing", destination: "/invoicing", permanent: true },
  ],
  headers: async () => [
    {
      source: "/(.*)",
      headers: [
        { key: "X-Frame-Options", value: "DENY" },
        { key: "X-Content-Type-Options", value: "nosniff" },
        { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
        { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=(self)" },
        {
          key: "Content-Security-Policy",
          value: `default-src 'self'; script-src 'self' 'unsafe-inline' https://geowidget.inpost.pl https://static.cloudflareinsights.com https://js.stripe.com; style-src 'self' 'unsafe-inline' https://geowidget.inpost.pl; img-src 'self' data: https: blob:; connect-src 'self' ${apiUrl} https://*.inpost.pl https://cloudflareinsights.com https://api.stripe.com https://*.sentry.io ${getWsDirectives()}; font-src 'self' data:; frame-src https://js.stripe.com https://hooks.stripe.com; frame-ancestors 'none'; base-uri 'self'; form-action 'self';`,
        },
      ],
    },
  ],
};

export default withSentryConfig(nextConfig, {
  // Upload source maps to Sentry during build (CI sets SENTRY_AUTH_TOKEN)
  silent: !process.env.CI,
  org: process.env.SENTRY_ORG,
  project: process.env.SENTRY_PROJECT,

  // Disable source map upload locally (only in CI)
  disableSourceMapUpload: !process.env.SENTRY_AUTH_TOKEN,

  // Automatically tree-shake Sentry logger in production
  disableLogger: true,

  // Hide source maps from the client (uploaded to Sentry only)
  hideSourceMaps: true,
});
```

**Step 2: Create global-error.tsx**

Create `apps/dashboard/src/app/global-error.tsx`:

```tsx
"use client";

import * as Sentry from "@sentry/nextjs";
import { useEffect } from "react";

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  return (
    <html>
      <body>
        <div style={{ padding: "2rem", textAlign: "center" }}>
          <h2>Wystąpił nieoczekiwany błąd</h2>
          <p style={{ color: "#666", marginTop: "0.5rem" }}>
            Błąd został automatycznie zgłoszony. Spróbuj ponownie.
          </p>
          <button
            onClick={reset}
            style={{
              marginTop: "1rem",
              padding: "0.5rem 1rem",
              cursor: "pointer",
            }}
          >
            Spróbuj ponownie
          </button>
        </div>
      </body>
    </html>
  );
}
```

**Step 3: Verify it builds**

```bash
cd apps/dashboard && npx next build 2>&1 | tail -10
```

**Step 4: Commit**

```bash
git add apps/dashboard/next.config.ts apps/dashboard/src/app/global-error.tsx
git commit -m "feat: wrap next.config with Sentry, add CSP and global error boundary"
```

---

### Task 10: Helm — Add Sentry env vars to API server deployment

**Files:**
- Modify: `deploy/helm/openoms/templates/api-server/deployment.yaml`
- Modify: `deploy/helm/openoms/templates/configmap.yaml`

**Step 1: Add SENTRY_DSN secret to deployment.yaml**

In `deploy/helm/openoms/templates/api-server/deployment.yaml`, after the STRIPE_WEBHOOK_SECRET block (line 131), add:

```yaml
            - name: SENTRY_DSN
              valueFrom:
                secretKeyRef:
                  name: {{ include "openoms.secretName" . }}
                  key: SENTRY_DSN
                  optional: true
```

**Step 2: Add Sentry config to configmap.yaml**

In `deploy/helm/openoms/templates/configmap.yaml`, after the BILLING_PLANS block (line 29), add:

```yaml
  {{- if .Values.apiServer.sentry }}
  SENTRY_ENVIRONMENT: {{ .Values.apiServer.sentry.environment | default "production" | quote }}
  SENTRY_TRACES_SAMPLE_RATE: {{ .Values.apiServer.sentry.tracesSampleRate | default "0.1" | quote }}
  {{- end }}
```

**Step 3: Verify Helm template renders**

```bash
cd deploy/helm/openoms && helm template . 2>&1 | head -20
```

If helm is not available locally, just verify YAML syntax:
```bash
cat deploy/helm/openoms/templates/api-server/deployment.yaml | head -5
```

**Step 4: Commit**

```bash
git add deploy/helm/openoms/templates/api-server/deployment.yaml deploy/helm/openoms/templates/configmap.yaml
git commit -m "feat: add Sentry env vars to Helm chart"
```

---

### Task 11: Helm — Add dashboard Sentry build arg

**Files:**
- Modify: `deploy/helm/openoms/templates/dashboard/deployment.yaml` (if exists)
- Modify: `apps/dashboard/Dockerfile`

**Step 1: Add SENTRY_DSN build arg to Dockerfile**

In `apps/dashboard/Dockerfile`, after the `NEXT_PUBLIC_INPOST_GEOWIDGET_TOKEN` block (line 22), add:

```dockerfile
ARG NEXT_PUBLIC_SENTRY_DSN
ENV NEXT_PUBLIC_SENTRY_DSN=${NEXT_PUBLIC_SENTRY_DSN}

ARG NEXT_PUBLIC_SENTRY_ENVIRONMENT
ENV NEXT_PUBLIC_SENTRY_ENVIRONMENT=${NEXT_PUBLIC_SENTRY_ENVIRONMENT}

ARG SENTRY_AUTH_TOKEN
ARG SENTRY_ORG
ARG SENTRY_PROJECT
```

**Step 2: Add build args to release.yml**

In `.github/workflows/release.yml`, in the `build-dashboard` job's build-args (line 73), add:

```yaml
          build-args: |
            NEXT_PUBLIC_API_URL=${{ vars.NEXT_PUBLIC_API_URL || 'http://localhost:8080' }}
            NEXT_PUBLIC_SENTRY_DSN=${{ secrets.NEXT_PUBLIC_SENTRY_DSN || '' }}
            NEXT_PUBLIC_SENTRY_ENVIRONMENT=production
            SENTRY_AUTH_TOKEN=${{ secrets.SENTRY_AUTH_TOKEN || '' }}
            SENTRY_ORG=${{ vars.SENTRY_ORG || '' }}
            SENTRY_PROJECT=${{ vars.SENTRY_PROJECT || '' }}
```

**Step 3: Commit**

```bash
git add apps/dashboard/Dockerfile .github/workflows/release.yml
git commit -m "feat: add Sentry build args to dashboard Dockerfile and CI"
```

---

### Task 12: Helm — Grafana Alloy deployment manifests

**Files:**
- Create: `deploy/helm/openoms/templates/monitoring/alloy-configmap.yaml`
- Create: `deploy/helm/openoms/templates/monitoring/alloy-deployment.yaml`
- Create: `deploy/helm/openoms/templates/monitoring/alloy-serviceaccount.yaml`

**Step 1: Create Alloy ConfigMap**

Create `deploy/helm/openoms/templates/monitoring/alloy-configmap.yaml`:

```yaml
{{- if .Values.monitoring.enabled }}
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "openoms.fullname" . }}-alloy-config
  namespace: {{ .Values.namespace }}
  labels:
    {{- include "openoms.labels" . | nindent 4 }}
data:
  config.alloy: |
    prometheus.scrape "openoms_api" {
      targets = [{
        __address__ = "{{ include "openoms.fullname" . }}-api.{{ .Values.namespace }}.svc.cluster.local:{{ .Values.apiServer.port }}",
      }]
      forward_to = [prometheus.remote_write.grafana_cloud.receiver]
      scrape_interval = "15s"
      metrics_path = "/metrics"
      bearer_token = env("METRICS_TOKEN")
    }

    prometheus.remote_write "grafana_cloud" {
      endpoint {
        url = env("GRAFANA_REMOTE_WRITE_URL")
        basic_auth {
          username = env("GRAFANA_USER")
          password = env("GRAFANA_TOKEN")
        }
      }
    }
{{- end }}
```

**Step 2: Create Alloy Deployment**

Create `deploy/helm/openoms/templates/monitoring/alloy-deployment.yaml`:

```yaml
{{- if .Values.monitoring.enabled }}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "openoms.fullname" . }}-alloy
  namespace: {{ .Values.namespace }}
  labels:
    {{- include "openoms.labels" . | nindent 4 }}
    app.kubernetes.io/component: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/component: alloy
  template:
    metadata:
      annotations:
        checksum/config: {{ include (print $.Template.BasePath "/monitoring/alloy-configmap.yaml") . | sha256sum }}
      labels:
        app.kubernetes.io/component: alloy
    spec:
      serviceAccountName: {{ include "openoms.fullname" . }}-alloy
      automountServiceAccountToken: false
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534
        fsGroup: 65534
      containers:
        - name: alloy
          image: grafana/alloy:v1.8.2
          args:
            - run
            - /etc/alloy/config.alloy
            - --storage.path=/tmp/alloy
          ports:
            - name: http
              containerPort: 12345
              protocol: TCP
          env:
            - name: METRICS_TOKEN
              valueFrom:
                secretKeyRef:
                  name: {{ include "openoms.secretName" . }}
                  key: METRICS_TOKEN
                  optional: true
            - name: GRAFANA_REMOTE_WRITE_URL
              valueFrom:
                secretKeyRef:
                  name: {{ include "openoms.fullname" . }}-monitoring
                  key: GRAFANA_REMOTE_WRITE_URL
            - name: GRAFANA_USER
              valueFrom:
                secretKeyRef:
                  name: {{ include "openoms.fullname" . }}-monitoring
                  key: GRAFANA_USER
            - name: GRAFANA_TOKEN
              valueFrom:
                secretKeyRef:
                  name: {{ include "openoms.fullname" . }}-monitoring
                  key: GRAFANA_TOKEN
          resources:
            requests:
              cpu: 50m
              memory: 64Mi
            limits:
              cpu: 200m
              memory: 128Mi
          securityContext:
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
            capabilities:
              drop: ["ALL"]
          volumeMounts:
            - name: config
              mountPath: /etc/alloy
              readOnly: true
            - name: tmp
              mountPath: /tmp
      volumes:
        - name: config
          configMap:
            name: {{ include "openoms.fullname" . }}-alloy-config
        - name: tmp
          emptyDir:
            sizeLimit: 50Mi
{{- end }}
```

**Step 3: Create ServiceAccount**

Create `deploy/helm/openoms/templates/monitoring/alloy-serviceaccount.yaml`:

```yaml
{{- if .Values.monitoring.enabled }}
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ include "openoms.fullname" . }}-alloy
  namespace: {{ .Values.namespace }}
  labels:
    {{- include "openoms.labels" . | nindent 4 }}
    app.kubernetes.io/component: monitoring
automountServiceAccountToken: false
{{- end }}
```

**Step 4: Commit**

```bash
git add deploy/helm/openoms/templates/monitoring/
git commit -m "feat: add Grafana Alloy deployment manifests for metrics forwarding"
```

---

### Task 13: Run full test suite + lint

**Files:** None (verification only)

**Step 1: Go tests**

```bash
cd apps/api-server && go test ./... 2>&1 | tail -10
```

**Step 2: Go lint**

```bash
cd apps/api-server && golangci-lint run ./... 2>&1 | tail -10
```

**Step 3: Dashboard lint**

```bash
cd apps/dashboard && npx eslint --quiet src/ 2>&1 | tail -10
```

**Step 4: Dashboard build**

```bash
cd apps/dashboard && npx next build 2>&1 | tail -10
```

**Step 5: Fix any issues found, then commit fixes**

```bash
git add -A && git commit -m "fix: address lint and test issues from Sentry integration"
```

---

## Post-Implementation (Manual / Enterprise)

These are NOT code tasks — they require manual setup in external services:

### Sentry Setup
1. Create Sentry auth token for CI (Settings → Auth Tokens → Create)
2. Add GitHub Secrets: `SENTRY_AUTH_TOKEN`, `NEXT_PUBLIC_SENTRY_DSN`, `SENTRY_ORG`, `SENTRY_PROJECT`
3. Add k8s secret: `SENTRY_DSN` in openoms-secret
4. Set up Discord alert integration in Sentry (Settings → Integrations → Discord)

### Grafana Cloud Setup
1. Create k8s secret `openoms-monitoring` with: `GRAFANA_REMOTE_WRITE_URL`, `GRAFANA_USER`, `GRAFANA_TOKEN`
2. Set `monitoring.enabled: true` in Helm values
3. Create dashboards in Grafana Cloud UI:
   - Import Prometheus dashboard (ID 3662 — customizable)
   - Create OpenOMS-specific panels: request rate, error rate, latency p50/p95/p99
4. Configure alert rules + Discord notification channel

---

## Verification Checklist

```bash
# Go builds and tests pass
cd apps/api-server && go build ./... && go test ./...

# Dashboard builds
cd apps/dashboard && npm run lint && npx next build

# Helm templates render without errors
cd deploy/helm/openoms && helm template . --set monitoring.enabled=true

# Local test: Start API with SENTRY_DSN set, trigger a panic endpoint → appears in Sentry
# Local test: Start dashboard with NEXT_PUBLIC_SENTRY_DSN set, throw error → appears in Sentry
```
