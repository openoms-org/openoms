# OPE-308 DPD InfoServices Tracking Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the permanent DPD tracking stub with real DPD InfoServices SOAP tracking while keeping DPD Services REST shipment/label behavior unchanged.

**Architecture:** DPD shipment creation and labels continue to use DPD Services REST. Tracking is implemented through the separate DPD InfoServices SOAP endpoint `DPDInfoServicesObjEvents`, using `getEventsForWaybillV1`. Existing DPD credentials default the InfoServices channel to `master_fid`; optional `info_login`, `info_password`, and `info_channel` credentials allow tenants to use separate InfoServices credentials when DPD assigns them.

**Tech Stack:** Go 1.25, `net/http`, `encoding/xml`, existing `dpd-go-sdk`, OpenOMS carrier provider interface, Next.js dashboard provider credential metadata.

---

## Scope And Sources

Official DPD source checked on 2026-05-17:

- DPD Polska Web Service page lists DPD InfoServices SOAP API as the module for retrieving statuses.
- DPD InfoServices documentation `INFO_Services_v2 - ENG.pdf` documents `getEventsForWaybillV1(waybill, eventsSelectType, language, authDataV1)` and production WSDL `https://dpdinfoservices.dpd.com.pl/DPDInfoServicesObjEventsService/DPDInfoServicesObjEvents?wsdl`.

This PR does not implement DPD customer-channel batch polling or `markEventsAsProcessedV1`; OpenOMS `TrackingPoller` already polls individual shipment tracking numbers, so the safe first implementation is per-waybill on-demand tracking.

## Files

- Modify: `packages/dpd-go-sdk/client.go`
- Modify: `packages/dpd-go-sdk/models.go`
- Modify: `packages/dpd-go-sdk/shipments.go`
- Modify: `packages/dpd-go-sdk/statusmap.go`
- Create: `packages/dpd-go-sdk/infoservices.go`
- Create: `packages/dpd-go-sdk/infoservices_test.go`
- Modify: `packages/dpd-go-sdk/client_test.go`
- Modify: `packages/dpd-go-sdk/spec_test.go`
- Modify: `apps/api-server/internal/integration/carriers/dpd.go`
- Modify: `apps/api-server/internal/integration/carriers/dpd_test.go`
- Modify: `apps/api-server/internal/integration/carriers/dpd_production_test.go`
- Modify: `apps/dashboard/src/lib/constants.ts`
- Modify: `apps/dashboard/messages/pl/integrations.json`
- Modify: `apps/dashboard/messages/en/integrations.json`
- Modify: `docs/system-documentation.md`

## Task 1: SDK SOAP Client

- [ ] Add failing SDK tests in `packages/dpd-go-sdk/infoservices_test.go`.

Test cases:

1. `TestInfoServicesGetTracking_SendsWaybillSOAPRequest`
   - Start `httptest.Server`.
   - Create client with `WithInfoServicesBaseURL(server.URL)` and `WithInfoServicesCredentials("info-user", "info-pass", "channel-1")`.
   - Call `client.Shipments.GetTracking(ctx, "0000012345678")`.
   - Assert request method is `POST`.
   - Assert SOAP body contains `getEventsForWaybillV1`, waybill, `ONLY_LAST`, `EN`, login, password, and channel.

2. `TestInfoServicesGetTracking_ParsesEventsChronologically`
   - Mock two SOAP events returned newest first.
   - Assert SDK returns two events oldest first so OpenOMS `TrackingPoller` can use the last event as newest.
   - Assert business code `040101`, description, depot name, country, waybill, and timestamp are parsed.

3. `TestInfoServicesGetTracking_ReturnsSOAPFault`
   - Mock a SOAP fault with `faultstring`.
   - Assert error contains the fault text.

4. `TestInfoServicesGetTracking_RequiresInfoChannel`
   - Create client with empty `masterFid` and no info channel.
   - Assert `GetTracking` fails before HTTP with a clear config error.

- [ ] Run RED:

```bash
cd packages/dpd-go-sdk
go test ./... -run 'TestInfoServicesGetTracking' -count=1
```

Expected: compile/test failures because InfoServices helpers do not exist yet.

- [ ] Implement `packages/dpd-go-sdk/infoservices.go`.

Implementation requirements:

- `productionInfoServicesBaseURL = "https://dpdinfoservices.dpd.com.pl/DPDInfoServicesObjEventsService/DPDInfoServicesObjEvents"`.
- Add client fields `infoServicesBaseURL`, `infoLogin`, `infoPassword`, `infoChannel`.
- Default `infoLogin/login`, `infoPassword/password`, and `infoChannel/masterFid`.
- Add options `WithInfoServicesBaseURL(url string)` and `WithInfoServicesCredentials(login, password, channel string)`.
- Build SOAP 1.1 envelope with namespace `http://events.dpdinfoservices.dpd.com.pl/`.
- Use `http.NewRequestWithContext`, `Content-Type: text/xml; charset=utf-8`, `SOAPAction: ""`, and bounded body read with `io.LimitReader`.
- Parse SOAP fault before parsing successful response.
- Parse `eventTime` layouts `2006-01-02T15:04:05.999`, `2006-01-02T15:04:05`, and `time.RFC3339Nano`, using `Europe/Warsaw` for DPD timestamps without an explicit offset and normalizing returned values to UTC.
- Reverse returned DPD events so OpenOMS receives chronological order.

- [ ] Update `packages/dpd-go-sdk/shipments.go`.

`ShipmentService.GetTracking` should call `s.client.getInfoServicesTracking(ctx, trackingNumber)` instead of returning the permanent unsupported error.

- [ ] Run GREEN:

```bash
cd packages/dpd-go-sdk
go test ./... -run 'TestInfoServicesGetTracking' -count=1
```

Expected: PASS.

## Task 2: DPD Status Mapping

- [ ] Add/adjust SDK status map tests.

Update `packages/dpd-go-sdk/client_test.go` and `packages/dpd-go-sdk/spec_test.go` to expect:

- `030103 -> label_ready`
- `040101 -> picked_up`
- `040102 -> picked_up`
- `050101 -> in_transit`
- `120100 -> in_transit`
- `160101 -> in_transit`
- `170101 -> out_for_delivery`
- `190101 -> delivered`
- `190104 -> delivered`
- `701901 -> delivered`
- `230403 -> returned`
- `230408 -> returned`

- [ ] Run RED:

```bash
cd packages/dpd-go-sdk
go test ./... -run 'Test.*Status' -count=1
```

Expected: numeric DPD InfoServices codes fail until the map is extended.

- [ ] Extend `packages/dpd-go-sdk/statusmap.go`.

Keep existing symbolic statuses for REST/webhook compatibility and add numeric InfoServices codes. Map known failure/return/pickup/delivery states conservatively; unknown codes must still return `ok=false`.

- [ ] Run GREEN:

```bash
cd packages/dpd-go-sdk
go test ./... -run 'Test.*Status' -count=1
```

Expected: PASS.

## Task 3: API Carrier Provider

- [ ] Add failing provider tests.

Modify `apps/api-server/internal/integration/carriers/dpd_test.go`:

- `newTestDPDProvider` should accept REST and InfoServices server URLs or configure both to the same server when not relevant.
- Add `TestDPD_GetTracking_MapsInfoServicesEvents`.
  - Mock InfoServices SOAP response with `040101` and `190101`.
  - Assert returned OpenOMS tracking events have status codes `040101` and `190101`, mapped timestamps, location, and details.

Modify `apps/api-server/internal/integration/carriers/dpd_production_test.go`:

- Replace `TestDPD_GetTracking_ReturnsUnsupportedError` with a test proving the provider calls InfoServices and returns parsed events.

- [ ] Run RED:

```bash
cd apps/api-server
go test ./internal/integration/carriers -run 'TestDPD_GetTracking' -count=1
```

Expected: FAIL because provider still returns unsupported error.

- [ ] Update `apps/api-server/internal/integration/carriers/dpd.go`.

Changes:

- Extend `DPDCredentials` with:
  - `InfoLogin string json:"info_login,omitempty"`
  - `InfoPassword string json:"info_password,omitempty"`
  - `InfoChannel string json:"info_channel,omitempty"`
- Build DPD SDK options:
  - always use safe HTTP client.
  - if optional InfoServices fields are present, call `dpdsdk.WithInfoServicesCredentials(...)`.
  - if only `info_channel` is present, reuse primary `login/password` with that channel.
  - if no InfoServices fields are present, SDK defaults to primary `login/password/master_fid`.
- Implement `GetTracking` by calling `p.client.Shipments.GetTracking`.
- Convert SDK events to `integration.TrackingEvent`.

- [ ] Run GREEN:

```bash
cd apps/api-server
go test ./internal/integration/carriers -run 'TestDPD_GetTracking' -count=1
```

Expected: PASS.

## Task 4: Dashboard Credential Fields

- [ ] Update DPD credential metadata in `apps/dashboard/src/lib/constants.ts`.

Add optional fields after `master_fid`:

- `info_channel`, optional text, help text explains it is DPD InfoServices channel and defaults to Master FID.
- `info_login`, optional text, help text explains it overrides the DPD Services login for InfoServices only.
- `info_password`, optional password, help text explains it overrides the DPD Services password for InfoServices only.

- [ ] Add Polish and English translations in:

- `apps/dashboard/messages/pl/integrations.json`
- `apps/dashboard/messages/en/integrations.json`

- [ ] Run targeted frontend validation:

```bash
cd apps/dashboard
npm run lint:quiet
```

Expected: PASS.

## Task 5: Docs And Validation

- [ ] Update `docs/system-documentation.md`.

Document DPD support as:

- shipment creation/labels: DPD Services REST,
- tracking: DPD InfoServices SOAP `getEventsForWaybillV1`,
- credentials: primary login/password/master FID, optional InfoServices login/password/channel.

- [ ] Run targeted package tests:

```bash
cd packages/dpd-go-sdk
go test ./... -count=1
```

Expected: PASS.

- [ ] Run targeted API carrier tests:

```bash
cd apps/api-server
go test ./internal/integration/carriers -count=1
```

Expected: PASS.

- [ ] Run self-review:

```bash
cd .
git diff --check
git diff --stat
```

Expected: no whitespace errors; diff limited to DPD SDK/provider/dashboard credential metadata/docs/plan.

- [ ] Run full public validation before push:

```bash
cd .
./scripts/local-ci.sh
```

Expected: PASS with `/tmp/openoms-local-ci-full-results.txt` matching current clean HEAD.

## Risks And Rollback

- Risk: DPD InfoServices credentials may differ from DPD Services REST credentials. Mitigation: optional override fields and default fallback to existing credentials.
- Risk: DPD event ordering is newest-first per docs. Mitigation: SDK returns chronological order so existing `TrackingPoller` last-event behavior remains correct.
- Risk: status code mapping may miss obscure InfoServices codes. Mitigation: map core pickup/in-transit/out-for-delivery/delivered/returned states now; unknown codes remain no-op rather than forcing wrong transitions.
- Rollback: revert this PR. Existing shipment creation/label behavior is isolated from InfoServices tracking changes.
