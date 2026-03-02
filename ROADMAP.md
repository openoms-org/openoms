# Roadmap

> Last updated: March 2026

## Done

- [x] Allegro integration (OAuth2, offers, orders, "Wysylam z Allegro" shipments)
- [x] InPost integration (Paczkomaty, courier, label generation, tracking, webhooks)
- [x] Dashboard with Kanban board, dark mode, and PWA support
- [x] Order, product, and customer management (CRUD, tags, custom fields, custom statuses)
- [x] Automation rules engine (triggers, conditions, actions, delays, schedules)
- [x] Packing station with barcode scanner (pick & pack workflow)
- [x] Reports, advanced analytics, and CSV export
- [x] 2FA/TOTP authentication, RBAC with custom roles
- [x] Audit log, outgoing webhooks (HMAC-SHA256)
- [x] Returns/RMA system (6-status lifecycle)
- [x] WebSocket real-time updates
- [x] OpenAPI 3.1 spec with Swagger UI
- [x] Prometheus metrics
- [x] Docker Compose (dev + prod) and Helm chart (k3s/k8s)
- [x] CI/CD with GitHub Actions (lint, test, security scan, Trivy)
- [x] E2E test suite (21 Playwright specs, 119 tests covering CRUD, lifecycle, auth)
- [x] CodeQL code quality (zero findings)
- [x] Database backup CronJob (S3, daily, 30-day retention)
- [x] Security hardening (Go 1.25, dependency audits, Helm NetworkPolicies)
- [x] Onboarding wizard (4-step guided setup for new tenants)
- [x] DHL, DPD, GLS carrier SDK audit and verification against official API docs
- [x] Erli marketplace SDK rebuild and verification
- [x] Supplier portal (public, token-based access for suppliers)
- [x] Supplier product enrichment, category mapping, and Allegro parameter mapping
- [x] Carrier rate shopping
- [x] Product CSV import with preview
- [x] SMS notifications (SMSAPI / Twilio)
- [x] Marketing (Mailchimp sync, campaigns)
- [x] Print templates (orders, invoices, labels)
- [x] Payment checkout integration for self-service registration
- [x] License token registration (Ed25519 JWT with replay protection)

## In Progress (code written, needs production testing)

- [ ] 7 additional carriers (DHL, DPD, GLS verified; UPS, Poczta Polska, Orlen Paczka, FedEx need testing)
- [ ] 10 marketplaces (Amazon, eBay, WooCommerce, Kaufland, OLX, Empik, Erli verified; Shoper, PrestaShop, Shopify need testing)
- [ ] Repricing engine (4 pricing strategies with simulation)
- [ ] Multi-warehouse with PZ/WZ/MM documents and stocktaking
- [ ] Invoicing (Fakturownia, inFakt, wFirma) + KSeF e-invoicing
- [ ] Product variants and bundles
- [ ] Multi-currency with NBP exchange rates
- [ ] Helpdesk (Freshdesk) integration

## Planned

- [ ] Public SaaS (hosted version with pricing tiers)
- [ ] Mobile application
- [ ] Plugin/extension marketplace
- [ ] Customer self-service portal (returns, tracking)
- [ ] AI demand forecasting
- [ ] BaseLinker data import
- [ ] Monitoring and alerting (Grafana / Sentry)

---

This roadmap is a living document. Priorities shift based on community feedback.

Have ideas? [Join our Discord](https://discord.gg/3Z5hzeH5) or [open an issue](https://github.com/openoms-org/openoms/issues).
