# SSRF and External Request Security Audit

**Audit Date:** 2026-03-03

## Executive Summary

The OpenOMS API server has a well-designed centralized SSRF protection mechanism (`netutil` package) that is consistently applied across all direct outgoing HTTP request paths in the core service layer. However, there are several **CRITICAL** findings related to marketplace/store SDK integrations that use user-controlled URLs without SSRF protection, and some secondary issues around redirect handling and response body limits.

---

## 1. Outgoing Webhook Dispatcher

**Files examined:**
- `apps/api-server/internal/service/webhook_dispatch_service.go`
- `apps/api-server/internal/netutil/private_ip.go`
- `apps/api-server/internal/handler/settings_handler.go` (lines 548-552, 1207-1218)

| Check | Rating | Details |
|-------|--------|---------|
| DNS resolution before request | **OK** | `NoPrivateDialer()` resolves DNS at connect time (line 73-83 of `private_ip.go`), then connects directly to the resolved IP (line 86), preventing TOCTOU attacks. |
| Blocks private IP ranges | **OK** | Comprehensive list of 21 CIDR ranges (lines 17-43 of `private_ip.go`): RFC 1918 (10/8, 172.16/12, 192.168/16), loopback (127/8), link-local (169.254/16 -- blocks cloud metadata), CGN (100.64/10), multicast, reserved, IPv6 equivalents (::1, fc00::/7, fe80::/10), and IPv4-mapped IPv6. |
| URL validation at save time | **OK** | `settings_handler.go` line 549: `netutil.IsPrivateURL(ep.URL)` blocks private URLs when saving webhook config. Also validated during settings import (line 1212). |
| Follows redirects | **WARNING** | The `http.Client` uses Go's default `CheckRedirect` policy (follows up to 10 redirects). **The `NoPrivateDialer` does re-check the redirect target** because every new TCP connection goes through the custom `DialContext`, so redirects to private IPs are blocked. This is correct. |
| Request timeout | **OK** | 10-second timeout on the HTTP client (line 52). Per-delivery context timeout also applies. |
| Response size limit | **WARNING** | Line 170: `io.ReadAll(resp.Body)` drains the response body without any size limit. A malicious webhook endpoint could return a multi-gigabyte response and exhaust server memory. |
| Retry logic | **OK** | Exponential backoff (1s, 4s, 16s) with max 3 retries. |
| HMAC signing | **OK** | HMAC-SHA256 signature sent via `X-Webhook-Signature` header. |

### Automation Engine Webhook Action

**File:** `apps/api-server/internal/automation/actions.go` (lines 126-131, 753-791)

| Check | Rating | Details |
|-------|--------|---------|
| SSRF protection | **OK** | Uses `netutil.SafeHTTPClient(10 * time.Second)` (line 129). URL is user-controllable (from automation rule params), but the SafeHTTPClient blocks private IPs. |
| Response drain | **OK** | `resp.Body.Close()` via defer (line 784) -- does not read body. |

---

## 2. Supplier Feed URL Fetching

**File:** `apps/api-server/internal/service/supplier_service.go` (lines 738-742, 824-828, 1462)

| Check | Rating | Details |
|-------|--------|---------|
| SSRF protection on feed fetch | **OK** | All three feed fetch paths use `netutil.SafeHTTPClient`: BTP XML catalogue and wizard (5 minutes for the ~200 MiB ProductCatalogue), IOF parser (60 seconds). |
| Cloud metadata attack | **OK** | `feed_url` set to `http://169.254.169.254/...` would be blocked by `NoPrivateDialer` (169.254.0.0/16 is in the blocked CIDRs list). |
| URL validation at save time | **WARNING** | When a supplier is created (line 140), the `feed_url` is stored directly without checking `IsPrivateURL`. The SSRF check happens at fetch time (via SafeHTTPClient), not at save time. This means a private URL will be stored but fail when fetched. Consider adding save-time validation for better UX. |
| Timeout | **OK** | 5-minute timeout on BTP XML catalogue fetches; 60-second timeout on IOF. |

**File:** `apps/api-server/internal/integration/btp/provider.go` (line 135)

| Check | Rating | Details |
|-------|--------|---------|
| BTP provider feed fetch | **OK** | Uses `netutil.SafeHTTPClient(5 * time.Minute)` for the XML catalogue only. REST catalogue/inventory keep the 30s client. |

---

## 3. Image Download/Redownload

**File:** `apps/api-server/internal/service/image_download_service.go`

| Check | Rating | Details |
|-------|--------|---------|
| SSRF protection | **OK** | Uses `netutil.SafeHTTPClient(30 * time.Second)` (line 39). |
| File size limit | **OK** | `http.MaxBytesReader(nil, resp.Body, 10<<20)` limits to 10 MB (line 215). |
| Content type validation | **WARNING** | Content type is read from the server's `Content-Type` header (line 208), defaulting to `image/jpeg` if empty (line 210). However, there is **no validation** that the content type is actually an image type. Any content type is accepted and passed through to storage. |
| Internal network scanning | **OK** | `SafeHTTPClient` blocks private IPs. |
| Per-request timeout | **OK** | 10-second per-download context timeout (line 188) + 30-second client timeout. |
| Local URL skip | **OK** | `isLocalURL()` (lines 247-258) skips URLs containing `/uploads/` or non-HTTP URLs. |

**File:** `apps/api-server/internal/handler/bg_removal_handler.go` (lines 279-309)

| Check | Rating | Details |
|-------|--------|---------|
| SSRF protection | **OK** | Uses `netutil.SafeHTTPClient(30 * time.Second)` (line 285). |
| URL scheme validation | **OK** | Validates scheme is `http` or `https` (line 281). |
| Size limit | **OK** | Uses `io.LimitReader(resp.Body, h.maxSize)` (line 302). |
| Content type validation | **OK** | Uses `http.DetectContentType()` on actual bytes (line 308), not server header. |

---

## 4. OAuth Callback URLs

**File:** `apps/api-server/internal/handler/allegro_auth_handler.go`

| Check | Rating | Details |
|-------|--------|---------|
| Redirect URI | **OK** | The `redirectURI()` (line 47-49) is **server-controlled**: `h.cfg.FrontendURL + "/marketplaces/allegro"`. The user cannot set an arbitrary redirect_uri. |
| State parameter | **OK** | Cryptographically random 16-byte state (lines 75-80), stored server-side with 10-minute expiry, consumed atomically on callback (loaded + deleted in one operation). |
| CSRF protection | **OK** | State is validated before exchanging the code. |

**File:** `apps/api-server/internal/handler/amazon_auth_handler.go`

| Check | Rating | Details |
|-------|--------|---------|
| OAuth flow | **OK** | Amazon uses a credential setup flow (not OAuth redirect). No redirect_uri involved. |

---

## 5. Other Outgoing HTTP Requests

### 5a. Store Integration Setup -- CRITICAL

**File:** `apps/api-server/internal/handler/store_auth_handler.go`

| Check | Rating | Details |
|-------|--------|---------|
| Shoper `shop_url` | **CRITICAL** | Line 51: `shopersdk.NewClient(body.ShopURL, ...)` -- user-provided `shop_url` is passed directly to the SDK, which uses `http.DefaultClient` (no SSRF protection). An attacker can set `shop_url` to `http://10.0.0.1:8080/` and scan the internal network. |
| PrestaShop `shop_url` | **CRITICAL** | Line 85: `prestashopsdk.NewClient(body.ShopURL, body.APIKey)` -- same issue. User-provided URL, SDK uses `http.DefaultClient`. |
| Shopify `shop_domain` | **WARNING** | Line 118: `shopifysdk.NewClient(body.ShopDomain, ...)` -- lower risk because the Shopify SDK normalizes the domain, but uses `http.DefaultClient`. |

### 5b. Marketplace Provider SDKs -- CRITICAL

| Provider | File | Rating | Details |
|----------|------|--------|---------|
| **Shoper** | `integration/shoper/provider.go:54` | **CRITICAL** | `shopersdk.NewClient(creds.ShopURL, ...)` -- uses `http.DefaultClient`. URL comes from user-set credentials. Every poll cycle (45s) makes requests to this URL. |
| **WooCommerce** | `integration/woocommerce/provider.go:54` | **CRITICAL** | `woocommercesdk.NewClient(creds.StoreURL, ...)` -- same issue. |
| **PrestaShop** | `integration/prestashop/provider.go:50` | **CRITICAL** | `prestashopsdk.NewClient(creds.ShopURL, creds.APIKey)` -- same issue. |
| **Mirakl** | `integration/mirakl/provider.go:48` | **CRITICAL** | `miraklsdk.NewClient(creds.BaseURL, creds.APIKey)` -- same issue. |
| **Allegro** | `integration/allegro/provider.go:55` | OK | Uses fixed Allegro API endpoints. URL not user-controllable. |
| **Amazon** | `integration/amazon/provider.go:53` | OK | Uses fixed Amazon SP-API endpoints. |
| **Kaufland** | `integration/kaufland/provider.go:55` | OK | Uses fixed endpoint. |
| **eBay** | `integration/ebay/provider.go:60` | OK | Uses fixed eBay API endpoints. |
| **OLX** | `integration/olx/provider.go:51` | OK | Uses fixed OLX API endpoint. |
| **Erli** | `integration/erli/provider.go:56` | OK | Uses fixed Erli API endpoint. |
| **Shopify** | `integration/shopify/provider.go:56` | **WARNING** | User-set `shop_domain`, partially normalized by SDK. |

### 5c. Freshdesk Service -- WARNING

**File:** `apps/api-server/internal/service/freshdesk_service.go`

| Check | Rating | Details |
|-------|--------|---------|
| Domain-based URL construction | **WARNING** | URLs constructed as `https://{domain}.freshdesk.com/...` where `domain` is user-provided. While constrained by the `.freshdesk.com` suffix, it could be used for credential exfiltration. |
| SSRF protection | **OK** | Uses `netutil.SafeHTTPClient(30 * time.Second)` (line 54). |
| Response size limit | **WARNING** | `io.ReadAll(resp.Body)` without size limit (line 105). |

### 5d. Mailchimp Service -- WARNING

**File:** `apps/api-server/internal/service/mailchimp_service.go`

| Check | Rating | Details |
|-------|--------|---------|
| SSRF protection | **OK** | Uses `netutil.SafeHTTPClient(30 * time.Second)` (line 45). |
| Response size limit | **WARNING** | `io.ReadAll(resp.Body)` without size limit (line 104). |

### 5e-5g. AI, Exchange Rate, BG Removal -- OK

All use hardcoded URLs with `SafeHTTPClient`. Not user-controllable.

### 5h. All 27 Go SDK Packages

**Path:** `packages/*/client.go`

| Check | Rating | Details |
|-------|--------|---------|
| Default HTTP client | **WARNING** | All 27 SDK packages default to `http.DefaultClient` (no timeout, no SSRF protection). They accept `WithHTTPClient()` option for injection, but **none of the integration providers pass a SafeHTTPClient**. This is the root cause of the CRITICAL findings above. |

---

## 6. Redirect Handling

| Check | Rating | Details |
|-------|--------|---------|
| Redirect following | **OK** | `NoPrivateDialer` operates at TCP `DialContext` level, so every redirect target's IP is re-checked. A redirect from a public IP to a private IP is blocked. |

---

## Summary of Findings

### CRITICAL (requires immediate fix)

1. **Store integration setup endpoints lack SSRF protection** (`store_auth_handler.go`). Shoper and PrestaShop setup handlers accept user-provided `shop_url` and make HTTP requests using `http.DefaultClient`.

2. **Marketplace SDK providers don't inject SafeHTTPClient** (all providers in `integration/`). For providers with user-controllable base URLs (Shoper, WooCommerce, PrestaShop, Mirakl), this enables persistent SSRF.

### WARNING (should fix)

3. **Webhook response body not size-limited** (`webhook_dispatch_service.go` line 170)
4. **Image download service does not validate content type** (`image_download_service.go`)
5. **Supplier feed_url not validated at save time** (`supplier_service.go`)
6. **Several services read response bodies without size limits**: freshdesk, mailchimp, ai
7. **Freshdesk domain not validated** (`freshdesk_service.go`)

### Recommended Fixes (Priority Order)

**P0 -- Critical (fix before next deploy):**

All SDK provider instantiations that accept user-controlled URLs must pass `SafeHTTPClient` via the `WithHTTPClient` option:

- `apps/api-server/internal/integration/shoper/provider.go` line 54
- `apps/api-server/internal/integration/woocommerce/provider.go` line 54
- `apps/api-server/internal/integration/prestashop/provider.go` line 50
- `apps/api-server/internal/integration/mirakl/provider.go` line 48
- `apps/api-server/internal/integration/shopify/provider.go` line 56
- `apps/api-server/internal/handler/store_auth_handler.go` lines 51, 85, 118

Example fix:
```go
import "github.com/openoms-org/openoms/apps/api-server/internal/netutil"

client := shopersdk.NewClient(creds.ShopURL, creds.ClientID, creds.ClientSecret,
    shopersdk.WithHTTPClient(netutil.SafeHTTPClient(30 * time.Second)),
)
```

**P1 -- Warning (fix soon):**

- Add `io.LimitReader` wrapper to webhook response drain (limit to 1MB)
- Validate image content type in `image_download_service.go`
- Add `IsPrivateURL` check when saving supplier feed URLs
- Add `io.LimitReader` to Freshdesk, Mailchimp, and AI service response reads
