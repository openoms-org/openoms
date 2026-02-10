# InPost ShipX API Reference

## Overview

InPost operates the dominant parcel locker network in Poland (Paczkomaty) and provides courier services. The platform is transitioning from the Legacy ShipX API to a new Global API. OpenOMS should implement the Legacy API first and prepare for migration.

## Base URLs

### Legacy ShipX API

| Environment | URL |
|---|---|
| Production | `https://api-shipx-pl.easypack24.net` |
| Sandbox | `https://sandbox-api-shipx-pl.easypack24.net` |

### Global API (2025+)

| Environment | URL |
|---|---|
| Production | `https://api.inpost-group.com` |
| Stage | `https://stage-api.inpost-group.com` |

## Authentication

### Legacy API

Bearer token obtained from the InPost Manager panel. No OAuth flow required — the token is generated in the web UI.

```
Authorization: Bearer {token}
Content-Type: application/json
```

### Global API

OAuth 2.1 with client credentials flow. Token expires in approximately 10 minutes (599 seconds).

```
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials&client_id={id}&client_secret={secret}
```

### Decision

Implement **Legacy API first** — it is stable, well-documented, and currently in production use. Prepare adapter interfaces so migration to Global API is a swap of the transport layer.

## Shipment Creation

```
POST /v1/organizations/{org_id}/shipments
```

### Required Fields

```json
{
  "receiver": {
    "name": "Jan Kowalski",
    "phone": "600123456",
    "email": "jan@example.com",
    "address": {
      "street": "Krakowska 1",
      "city": "Warszawa",
      "post_code": "00-001",
      "country_code": "PL"
    }
  },
  "parcels": [
    {
      "template": "small",
      "weight": {
        "amount": 2.5,
        "unit": "kg"
      }
    }
  ],
  "service": "inpost_locker_standard",
  "custom_attributes": {
    "target_point": "KRA012"
  }
}
```

### Services

| Service | Description |
|---|---|
| `inpost_locker_standard` | Standard parcel locker delivery |
| `inpost_courier_standard` | Standard courier delivery |

For locker deliveries, `custom_attributes.target_point` is required (locker machine code).

## Label Generation

```
POST /v1/shipments/labels
```

| Parameter | Type | Description |
|---|---|---|
| `shipment_ids` | int[] | List of shipment IDs (max 100) |
| `format` | string | PDF, EPL, EPL2, ZPL |
| `type` | string | `normal` (A6) or `A4` |

Returns binary label data. For bulk requests (>10 shipments), the response may be asynchronous — poll the returned URL until ready.

## Parcel Sizes

| Template | Height | Width | Length | Max Weight |
|---|---|---|---|---|
| A (small) | 8 cm | 38 cm | 64 cm | 25 kg |
| B (medium) | 19 cm | 38 cm | 64 cm | 25 kg |
| C (large) | 41 cm | 38 cm | 64 cm | 25 kg |

Alternatively, provide explicit `dimensions` (height, width, length in mm) and `weight` instead of a template.

## Webhooks

### Configuration

Register webhook URLs in the InPost Manager panel or via API.

### Events

| Event | Description |
|---|---|
| `shipment_status_changed` | Shipment status updated |
| `shipment_confirmation` | Shipment confirmed and label ready |
| `error` | Shipment processing error |

### Verification

Webhook requests include a signature header for verification:

```
X-InPost-Signature: {hmac_sha256_signature}
```

Compute HMAC-SHA256 of the raw request body using the webhook secret and compare with the header value.

### Requirements

- Must respond with HTTP 200 within **5 seconds**.
- Failed deliveries are retried with exponential backoff.
- After persistent failures, the webhook is deactivated.

## Status Mapping

| InPost Status | OpenOMS `shipment_status` | Order Transition |
|---|---|---|
| `created` | `created` | — |
| `offers_prepared` | `created` | — |
| `offer_selected` | `created` | — |
| `confirmed` | `label_ready` | processing -> ready_to_ship |
| `dispatched_by_sender` | `picked_up` | ready_to_ship -> shipped |
| `collected_from_sender` | `in_transit` | shipped -> in_transit |
| `taken_by_courier` | `in_transit` | — |
| `adopted_at_source_branch` | `in_transit` | — |
| `sent_from_source_branch` | `in_transit` | — |
| `adopted_at_sorting_center` | `in_transit` | — |
| `sent_from_sorting_center` | `in_transit` | — |
| `out_for_delivery` | `in_transit` | — |
| `ready_to_pickup` | `in_transit` | — |
| `delivered` | `delivered` | in_transit -> delivered |
| `returned_to_sender` | `returned` | — |
| `avizo` | `in_transit` | — |
| `claimed` | `delivered` | — |

## Sandbox Notes

- **Tracking is disabled** in sandbox — status transitions do not happen automatically.
- Use sandbox to test shipment creation and label generation only.
- Webhooks can be tested by manually triggering status changes in the sandbox panel.
- Sandbox `org_id` is different from production — configure per environment.

## Implementation Notes

1. **Organization ID**: Store `org_id` in integration settings per tenant. Required for all shipment endpoints.
2. **Label generation**: May be async for bulk requests. Implement polling with timeout for label readiness.
3. **Legacy first**: Implement against Legacy ShipX API. Define a `ShipmentProvider` interface so the Global API can be swapped in later.
4. **Webhook processing**: Enqueue webhook payloads into the job queue immediately, process asynchronously. Respond with 200 before processing to meet the 5-second requirement.
5. **Locker selection**: Consider integrating the InPost points lookup API (`GET /v1/points`) to let users select a target locker.
