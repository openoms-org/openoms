# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- **Erli marketplace**: HTTP handler (`erli_listings_handler.go`) for creating product listings on Erli.pl with HTML sanitization
- **Erli marketplace**: background order poller (`erli_order_poller.go`) registered alongside other marketplace pollers (Allegro, Amazon, WooCommerce, etc.)
- **erli-go-sdk**: integration tests against the Erli sandbox API (`integration_test.go`, build tag `integration`)
- **CI**: `erli-go-sdk` added to lint, govulncheck, vet, and auto-format jobs
- **Dashboard**: UPS, FedEx, GLS, Poczta Polska, and Orlen Paczka carrier-specific fields in the shipment form (service type selectors, insurance field for FedEx)

### Changed

- **erli-go-sdk**: `Offers.Create` now returns the created offer ID (`CreateOfferResponse.ID`) instead of discarding it
- **erli-go-sdk**: orders query constructed with `url.Values` instead of string concatenation
- **erli-go-sdk**: HTTP client uses a 30-second timeout (replaces `http.DefaultClient` with no timeout)
- **erli-go-sdk**: response body reads capped at 50 MB; error body reads capped at 1 MB to prevent memory exhaustion
- **Migration 000006**: `idx_message_templates_tenant_id` index and dynamic `GRANT` for `openoms_app`/`openoms` roles moved here from migration 000007 (index belongs with the table that creates it)
- **Automation**: template variable substitution now scoped to variables declared in the template's `Variables` list, preventing undeclared keys from being resolved

### Fixed

- `PushOffer` returned an empty string as the external ID, causing `NULL external_id` in `product_listings` and breaking stock/price sync for Erli
- `message_templates` tenant index was created in migration 000007 instead of 000006 where the table is defined
