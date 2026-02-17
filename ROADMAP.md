# Roadmap

> Last updated: February 2026

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

## In Progress (code written, needs production testing)

- [ ] 7 additional carriers (DHL, DPD, GLS, UPS, Poczta Polska, Orlen Paczka, FedEx)
- [ ] 10 marketplaces (Amazon, eBay, WooCommerce, Kaufland, OLX, Empik, Erli, Shoper, PrestaShop, Shopify)
- [ ] Repricing engine (4 pricing strategies with simulation)
- [ ] Multi-warehouse with PZ/WZ/MM documents and stocktaking
- [ ] Invoicing (Fakturownia, inFakt, wFirma) + KSeF e-invoicing
- [ ] Product variants and bundles
- [ ] Multi-currency with NBP exchange rates
- [ ] Carrier rate shopping
- [ ] Product CSV import with preview
- [ ] SMS notifications (SMSAPI / Twilio)
- [ ] Marketing (Mailchimp) and helpdesk (Freshdesk) integrations
- [ ] Print templates (orders, invoices, labels)

## Planned

- [ ] Public SaaS (hosted version with pricing tiers)
- [ ] Mobile application
- [ ] Plugin/extension marketplace
- [ ] Customer self-service portal (returns, tracking)
- [ ] Supplier portal
- [ ] AI demand forecasting

---

This roadmap is a living document. Priorities shift based on community feedback.

Have ideas? [Join our Discord](https://discord.gg/3Z5hzeH5) or [open an issue](https://github.com/openoms-org/openoms/issues).
