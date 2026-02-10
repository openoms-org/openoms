# Allegro REST API Reference

## Overview

Allegro.pl is the largest online marketplace in Poland and Central Europe. Integration with Allegro allows OpenOMS to import orders, synchronize stock, and manage offers across the platform.

## Base URLs

| Environment | URL |
|---|---|
| Production | `https://api.allegro.pl` |
| Sandbox | `https://api.allegro.pl.allegrosandbox.pl` |

## Authentication

Allegro uses **OAuth 2.0 Authorization Code Grant** for user-context operations and **Client Credentials** for public data access.

### OAuth 2.0 Endpoints

| Endpoint | URL |
|---|---|
| Authorization | `https://allegro.pl/auth/oauth/authorize` |
| Token | `https://allegro.pl/auth/oauth/token` |

### Token Lifetimes

| Token | Lifetime |
|---|---|
| Access token | 12 hours (43,199 seconds) |
| Refresh token | ~3 months |

### Headers

All API requests require:

```
Authorization: Bearer {access_token}
Accept: application/vnd.allegro.public.v1+json
Content-Type: application/vnd.allegro.public.v1+json
```

### Scopes

| Scope | Description |
|---|---|
| `allegro:api:orders:read` | Read order data |
| `allegro:api:orders:write` | Update order status |
| `allegro:api:sale:offers` | Manage sale offers |
| `allegro:api:sale:offers:read` | Read offer data |
| `allegro:api:profile:read` | Read user profile |

## Orders API

### List Orders

```
GET /order/checkout-forms
```

Returns orders from the last 12 months.

| Parameter | Type | Description |
|---|---|---|
| `limit` | int | Results per page (1-100, default 25) |
| `offset` | int | Pagination offset |
| `status` | string | Filter by status |
| `fulfillment.status` | string | Filter by fulfillment |
| `updatedAt.gte` | datetime | Updated after (ISO 8601) |

### Get Order Detail

```
GET /order/checkout-forms/{id}
```

Returns full order with buyer, delivery address, payment info, invoice details, and line items.

### Order Event Stream (Recommended)

```
GET /order/events
```

Polling-based event stream — the recommended method for tracking order changes. Preferred over webhooks for order data.

| Parameter | Type | Description |
|---|---|---|
| `from` | string | Last event ID for incremental polling |
| `type` | string[] | Filter by event type |

**Event Types:**

| Event | Description |
|---|---|
| `BOUGHT` | Order placed by buyer |
| `FILLED_IN` | Buyer filled in checkout form |
| `READY_FOR_PROCESSING` | Payment confirmed, ready to fulfill |
| `AUTO_CANCELLED` | System cancelled (e.g. unpaid) |
| `BUYER_CANCELLED` | Buyer cancelled the order |
| `FULFILLMENT_STATUS_CHANGED` | Shipment or fulfillment updated |

Events are available for the **last 24 hours** only. Store `last_event_id` persistently and poll regularly to avoid gaps.

## Offers / Products API

### List Offers

```
GET /sale/offers
```

| Parameter | Type | Description |
|---|---|---|
| `limit` | int | Results per page (1-60) |
| `offset` | int | Pagination offset |
| `name` | string | Filter by offer name |
| `publication.status` | string | ACTIVE, INACTIVE, ENDED |

### Get Offer Detail

```
GET /sale/product-offers/{offerId}
```

Returns full offer data including pricing, stock, parameters, and images.

## Webhooks

Webhooks have **limited support for orders** — use the event polling endpoint as the primary mechanism.

Webhooks are available for offer lifecycle events:

| Event | Description |
|---|---|
| `OFFER_ACTIVATED` | Offer went live |
| `OFFER_CHANGED` | Offer details modified |
| `OFFER_ENDED` | Offer ended or sold out |

## Rate Limits

| Metric | Value |
|---|---|
| Global limit | 9,000 requests/min per Client ID |
| Exceeded response | HTTP 429 with `Retry-After` header |

**Recommended client-side strategy:** Token-bucket algorithm at ~150 requests/min with burst capacity of 5.

## Data Mapping

| Allegro Field | OpenOMS Field | Notes |
|---|---|---|
| `checkout-form.id` | `external_id` | UUID, used for deduplication |
| `buyer.login` | `customer_name` | Allegro username |
| `buyer.email` | `customer_email` | |
| `delivery.address` | `shipping_address` | Flatten struct |
| `payment.paidAmount.amount` | `total_amount` | Decimal string |
| `payment.paidAmount.currency` | `currency` | Always PLN on allegro.pl |
| `lineItems[].offer.id` | `line_items[].external_offer_id` | |
| `lineItems[].quantity` | `line_items[].quantity` | |
| `lineItems[].price.amount` | `line_items[].unit_price` | |
| `status` | `order_status` | Map to OpenOMS statuses |
| `fulfillment.status` | `fulfillment_status` | PROCESSING, SENT, etc. |
| `invoice.required` | `invoice_requested` | Boolean |

## 2025 Changes

- Stricter API access policies — new apps require formal review before production access.
- Financial penalties for API abuse and non-compliant integrations.
- Sandbox testing now mandatory before production approval.
- Rate limits enforced more aggressively with shorter ban windows.

## Implementation Notes

1. **Polling interval**: Poll `/order/events` every 15 minutes. Store last event ID in the database.
2. **Token refresh**: Refresh access tokens every 30 minutes (well before 12h expiry) to avoid gaps.
3. **Deduplication**: Use `checkout-form.id` as `external_id`. Check for existence before inserting.
4. **Error handling**: On HTTP 429, respect `Retry-After`. On 5xx, exponential backoff starting at 1s.
5. **Sandbox first**: Develop and test against sandbox environment before requesting production access.

## Links

- Developer Portal: https://developer.allegro.pl
- REST API Documentation: https://developer.allegro.pl/documentation
- OpenAPI Specification: https://developer.allegro.pl/swagger.yaml
- GitHub (examples): https://github.com/allegro/allegro-api
