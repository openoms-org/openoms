# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### Payment Checkout Integration
- **New API endpoints** (public, no auth required):
  - `GET /v1/billing/plans` — list available subscription plans (rate limit 60/min)
  - `POST /v1/billing/checkout` — create payment checkout session (rate limit 10/min)
  - `GET /v1/billing/checkout/{session_id}` — check checkout session status
- **Webhook handler**: `POST /v1/webhooks/stripe` with signature verification for payment event processing (checkout completed, subscription updates, payment failures)
- **Registration flow**: Added `checkout_session_id` to registration request for payment-verified signups
- **Database tables**: `billing_customers`, `billing_subscriptions`, `billing_checkout_sessions` with SECURITY DEFINER functions for pre-registration operations
- **Frontend**: Plan selection page at `/register`, post-payment form at `/register/complete`, invite-based registration moved to `/register/invite`
- **Configuration**: Plans loaded from `BILLING_PLANS` environment variable (JSON); payment disabled when API keys not configured

#### License Token Registration
- **Registration flow**: Added `license_token` to registration request for token-verified signups
- **Database**: `used_license_tokens` table with JTI-based replay protection
- **Validation**: Ed25519 signature verification via `LICENSE_PUBLIC_KEY` environment variable
- **SECURITY DEFINER**: `validate_license_token()` function for pre-registration token validation

#### Shipments Fix
- Fixed NULL scan error when querying shipments with nullable columns (pointer types for optional fields)

#### Onboarding Status Guard
- Added `isAuthenticated` guard to `useOnboardingStatus` hook preventing 401 errors on initial page load

#### Onboarding Wizard - First-Time Setup After Registration
- **New API Endpoints**:
  - `GET /v1/onboarding/status` — Retrieve current onboarding state (company setup, warehouses, integrations, team)
  - `PUT /v1/onboarding/step/{step}` — Mark a wizard step (1-4) as completed or skipped (admin only)
  - `POST /v1/onboarding/complete` — Mark entire onboarding as done (admin only)
  - `GET/PUT /v1/settings/onboarding` — Get/update onboarding settings (admin only)
- **Onboarding State Tracking**: Extended `OnboardingSettings` model with multi-step tracking (current step, completed steps, skipped steps, completion timestamp)
- **Dashboard Onboarding Page** (`/onboarding`): 4-step interactive wizard with:
  - Step 1: Company details (name, NIP, address, phone, email) — required
  - Step 2: Default warehouse setup (name, address, mark as default) — optional
  - Step 3: First integration (carrier or marketplace provider with credentials) — optional
  - Step 4: Team invitations (invite 1-2 members by email) — optional
- **Auto-redirect Logic**: New tenants redirected to `/onboarding` after login until setup is complete
- **Wizard Features**: Progress stepper, skip functionality (except step 1), "Finish later" button to save progress, completion screen with confirmation
- **Custom Hook**: `useOnboardingWizard` for managing multi-step form state and API calls
- **Database**: No schema migration required — onboarding state stored in `tenants.settings` JSONB under `onboarding` key; backwards-compatible with existing tenants

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
