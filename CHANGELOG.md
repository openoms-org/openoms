# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

#### Carrier SDK Audit & Corrections (DHL, DPD, GLS)
- **GLS SDK & Integration**: Corrected authentication from Bearer token to Basic Auth per ShipIT REST API spec; migrated tracking retrieval from GET to POST; updated cancel shipment from DELETE to POST; aligned response models with actual API structure; added COD amount propagation to shipment creation
- **DPD SDK & Integration**: Replaced fictional endpoints with official DPD REST API (dpdservices.dpd.com.pl/api/v1); corrected authentication flow using session-based tokens; implemented two-phase label retrieval (creation response + separate file fetch); added support for COD (cash on delivery) and insurance fields in shipment requests
- **DHL SDK & Integration**: Replaced fictional REST API with DHL24 SOAP WebAPI2 (dhl24.com.pl/webapi2); converted HTTP client to SOAP envelope marshaling; corrected service types (AH, 09, 12, EK, PI per DHL24 documentation); added support for domestic and international shipping; implemented proper SOAP response parsing with namespace handling
- **Frontend Carrier Fields**: Added missing COD/Insurance form fields for DPD shipments; corrected DHL service type options from arbitrary strings to valid DHL24 codes with proper labels
- **Specification Tests**: Added comprehensive test suites for all three carriers (GLS, DPD, DHL) verifying SDK responses match official API documentation and integration layer correctly propagates data

#### Erli SDK & Integration
- **Base URL correction**: Updated Erli API base URL from `https://api.erli.pl/v2` to `https://erli.pl/svc/shop-api` (official API endpoint per [Erli Shop API documentation](https://erli.pl/svc/shop-api/doc/))
- **Product endpoints**: Migrated from `/offers` endpoints to `/products/{externalId}` with seller SKU as required URL path parameter
- **Order status mapping**: Aligned with Erli's 3 actual order statuses — replaced 6 fictional mappings with: `pending` (unpaid), `purchased` (paid/COD), `cancelled`
- **Order polling**: Fixed status filter from `paid` to `purchased`; fixed pagination parameter from `cursor` to `after`
- **Asynchronous product creation**: Added handling for HTTP 202 Accepted responses with `ErrProductPendingValidation` sentinel; products now queued for validation on Erli side
- **Provider integration**: `PushOffer` now requires product SKU passed as `externalId` in URL path, not request body
- **Security**: Fixed sandbox URL configuration to prevent silent fallback to production endpoint

#### Carrier Service Types
- FedEx: Fixed `INTERNATIONAL_ECONOMY` to use proper `FEDEX_INTERNATIONAL_ECONOMY` prefix per REST API 2024+ requirements
- FedEx: Removed `FEDEX_GROUND` option (only available in Canada, not EU/Poland)
- UPS: Corrected label for service code 65 from "Express Saver" to "Worldwide Saver"
- GLS: Added service type selector with options (Standard, Express 10:00, Express 12:00) and wired through backend
- Poczta Polska: Extended service type options with POCZTEX_2_0, PACZKA_POCZTOWA, and EMS (Express Mail Service)

### Changed
- **SDK method signatures**: `Create()`, `UpdateStock()`, `UpdatePrice()` now require `externalID` (seller SKU) as explicit parameter
- **API client configuration**: Sandbox base URL now configurable via `WithBaseURL()` option instead of hardcoded value
