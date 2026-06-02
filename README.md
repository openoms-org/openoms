# OpenOMS

[![License: ELv2](https://img.shields.io/badge/License-ELv2-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)](https://go.dev/)
[![Build](https://img.shields.io/github/actions/workflow/status/openoms-org/openoms/ci.yml?branch=main&label=CI)](https://github.com/openoms-org/openoms/actions)
[![Discord](https://img.shields.io/discord/1234567890?color=5865F2&label=Discord&logo=discord&logoColor=white)](https://discord.gg/3Z5hzeH5)

**Open-source Order Management System for e-commerce.**

OpenOMS is a self-hostable, multi-tenant OMS with 463 API endpoints, 141 dashboard pages, and integrations with 8 marketplaces and 8 carriers. Built with Go and Next.js, designed for teams that need full control over their order operations.

> **Status: Active Development** — Looking for beta testers! [Join our Discord](https://discord.gg/3Z5hzeH5)

<!-- ![OpenOMS Dashboard](docs/screenshot.png) -->

---

## Features

### Order Management
- Custom order statuses, custom fields, and tags
- Order merge and split
- Kanban board view
- Barcode scanning (packing station)
- Automation rules engine (trigger, conditions, actions)
- Print templates (orders, invoices, shipping labels)
- CSV import/export
- Action delays (scheduled automation)

### Products and Inventory
- Product variants and bundles
- B2B pricing tiers
- Multi-warehouse stock management
- Warehouse documents (PZ/WZ/MM)
- Stocktaking (inventory counts)
- Product CSV import with preview
- Product categories with color badges

### Marketplace Integrations (11)

| Integration | Description | Status |
|---|---|---|
| **Allegro** | OAuth2, full offer management, listing creation, "Wysylam z Allegro" shipments | Verified |
| **Amazon SP-API** | Orders and catalog sync | In Development |
| **WooCommerce** | Bidirectional order and product sync | In Development |
| **eBay** | Order import and listing management | In Development |
| **Kaufland** | Marketplace integration | In Development |
| **OLX** | Listing and order management | In Development |
| **Mirakl / Empik** | Marketplace connector | In Development |
| **Erli** | Polish marketplace integration | In Development |
| **Shoper** | Polish e-commerce platform integration | In Development |
| **PrestaShop** | Open-source e-commerce platform | In Development |
| **Shopify** | SaaS e-commerce platform | In Development |

### Carrier Integrations (8)

| Integration | Description | Status |
|---|---|---|
| **InPost** | Paczkomaty (parcel lockers) + courier + webhooks + dispatch orders | Verified |
| **DHL** | Domestic and international shipping (DHL24 SOAP WebAPI2) | Verified |
| **DPD** | Parcel shipping (REST API with session-based auth) | Verified |
| **GLS** | Parcel shipping (ShipIT REST API with Basic Auth) | Verified |
| **UPS** | Domestic and international shipping | In Development |
| **Poczta Polska** | National postal service | In Development |
| **Orlen Paczka** | Parcel lockers | In Development |
| **FedEx** | International shipping | In Development |

Carrier rate shopping across all providers.

### Other Integrations

| Integration | Description | Status |
|---|---|---|
| **Fakturownia** | Invoice generation | In Development |
| **KSeF** | Polish national e-invoicing system | In Development |
| **Mailchimp** | Marketing automation | In Development |
| **Freshdesk** | Helpdesk tickets | In Development |
| **SMSAPI / Twilio** | SMS notifications | In Development |
| **OpenAI** | AI product categorization and descriptions | Verified |
| **NBP** | Exchange rates (multi-currency support) | Verified |

### Platform
- Multi-tenant SaaS with PostgreSQL Row-Level Security
- 463 REST API endpoints with OpenAPI 3.1 spec (Swagger UI)
- 141 dashboard pages with dark mode, PWA support, keyboard shortcuts
- **Guided onboarding wizard** for new tenants (company setup, warehouse, integration, team)
- Registration with invite tokens, license tokens, or payment checkout
- Background workers (21 registered: order pollers, stock sync, tracking, automation)
- RBAC with custom roles
- 2FA / TOTP authentication
- WebSocket real-time updates
- Outgoing webhooks (HMAC-SHA256 signed)
- Audit log
- Self-service returns portal
- Multi-currency with NBP exchange rates
- Customer management (CRM)
- Prometheus metrics (Bearer token auth)
- Security headers (CSP, X-Frame-Options, HSTS, Referrer-Policy)
- Kubernetes secrets encryption at rest, audit logging

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.25, chi/v5 router, pgx/v5 |
| Frontend | Next.js 16, React 19, TypeScript |
| Styling | Tailwind CSS v4, shadcn/ui |
| State / Data | Zustand, React Query, Zod v4 |
| Charts | Recharts |
| Database | PostgreSQL 16 (Row-Level Security) |
| Cache / Queue | Redis 7, asynq |
| Auth | Ed25519 JWT, bcrypt, TOTP |
| API Spec | OpenAPI 3.1, Swagger UI |
| E2E Tests | Playwright (22 specs, 124 tests) |
| CI/CD | GitHub Actions (lint, test, security scan, format verification, Trivy) |
| Deployment | Docker Compose (dev + prod), Helm chart (k3s/k8s) |
| Monitoring | Prometheus metrics (token-protected) |

### Codebase at a Glance

| Metric | Count |
|---|---|
| Go source files | 345 (121 test files) |
| TypeScript / TSX files | 308 |
| SQL migrations | 28 (56 up/down files) |
| API endpoints | 463 |
| Dashboard pages | 141 |
| React components | 96 |
| Custom hooks | 73 |
| Handlers / Services / Repos | 85 / 67 / 43 |
| Background workers | 21 |
| Middleware | 16 |
| SDK packages | 27 |

---

## Quick Start (Development)

**Prerequisites:** Go 1.25+, Node.js 22+, Docker with Docker Compose, [Task](https://taskfile.dev), [golang-migrate](https://github.com/golang-migrate/migrate)

```bash
# 1. Clone
git clone https://github.com/openoms-org/openoms.git
cd openoms

# 2. Configure environment
cp .env.example .env

# 3. Install dashboard dependencies, start infrastructure, run migrations, seed data
task setup

# 4. Start the API server and dashboard
task dev
```

Verify the API and dashboard are running:

```bash
curl http://localhost:8080/health
# Open http://localhost:3000/login
```

Seed login for local development:

| Field | Value |
|---|---|
| Organization | `dev` |
| Email | `admin@dev.local` |
| Password | `password123` |

This account is created only by `task setup` or `task seed`. Production and other self-hosted deployments should create their own administrator account or use the configured registration/invite flow; no default production password is provided.

### Development Commands

```bash
task setup       # Full setup: dashboard deps + containers + migrations + seed
task up          # Start PostgreSQL + Redis containers
task down        # Stop containers
task run         # Start API server (port 8080)
task dashboard   # Start dashboard dev server (port 3000)
task dev         # Start API server + dashboard in parallel
task dashboard:install  # Install dashboard dependencies
task migrate     # Run database migrations
task seed        # Load test data
task test        # Run all tests (race detection + coverage)
task lint        # Run golangci-lint on all modules
task fmt         # Format all Go source files
task clean       # Stop containers and remove volumes
```

---

## Repository Structure

```
openoms/
├── apps/
│   ├── api-server/              # Go backend (ELv2)
│   │   ├── cmd/server/          # Entrypoint
│   │   ├── internal/            # Handlers, services, repositories, workers
│   │   └── migrations/          # 28 migrations (56 SQL files)
│   └── dashboard/               # Next.js frontend (ELv2)
│       └── src/
├── packages/                    # 27 standalone SDK libraries (MIT)
├── deploy/
│   └── helm/openoms/          # Helm chart for k3s/k8s
├── docs/
│   └── system-documentation.md
├── docker-compose.dev.yml
├── docker-compose.prod.yml
├── Taskfile.yml
├── .github/workflows/
│   ├── ci.yml                 # Lint, test, security scan, format checks
│   └── release.yml            # CI/CD workflow that builds Docker images
└── .env.example
```

---

## SDK Packages

All packages are independently usable Go libraries, licensed under MIT.

| Package | Wraps |
|---|---|
| `allegro-go-sdk` | Allegro REST API (OAuth2, offers, orders, deliveries) |
| `amazon-sp-sdk` | Amazon Selling Partner API |
| `woocommerce-go-sdk` | WooCommerce REST API |
| `ebay-go-sdk` | eBay Browse / Sell APIs |
| `kaufland-go-sdk` | Kaufland Marketplace API |
| `olx-go-sdk` | OLX Partner API |
| `mirakl-go-sdk` | Mirakl Marketplace Platform API |
| `erli-go-sdk` | Erli Marketplace API |
| `inpost-go-sdk` | InPost ShipX API (shipments, points, tracking) |
| `dhl-go-sdk` | DHL Parcel API |
| `dpd-go-sdk` | DPD Web Services |
| `gls-go-sdk` | GLS Web API |
| `ups-go-sdk` | UPS Shipping / Tracking APIs |
| `poczta-polska-go-sdk` | Poczta Polska e-Nadawca API |
| `orlen-paczka-go-sdk` | Orlen Paczka API |
| `fedex-go-sdk` | FedEx Ship / Track APIs |
| `fakturownia-go-sdk` | Fakturownia Invoicing API |
| `ksef-go-sdk` | KSeF (Polish National e-Invoicing System) |
| `smsapi-go-sdk` | SMSAPI SMS Gateway |
| `order-engine` | Order state machine and domain events |
| `iof-parser` | IOF product feed parser |
| `shoper-go-sdk` | Shoper REST API |
| `prestashop-go-sdk` | PrestaShop Web Services |
| `shopify-go-sdk` | Shopify Admin API |
| `infakt-go-sdk` | inFakt Invoicing API |
| `wfirma-go-sdk` | wFirma Invoicing API |
| `btp-go-sdk` | BTP.pro Supplier API |

---

## Documentation

Full system documentation is available at [`docs/system-documentation.md`](docs/system-documentation.md).

The API server exposes an interactive OpenAPI 3.1 specification via Swagger UI at `/swagger/` when running.

---

## Deployment

### Docker Compose (Production)

```bash
cp .env.example .env
# Edit .env with production values (strong secrets, real credentials)

docker-compose -f docker-compose.prod.yml up -d --build
```

The production compose file includes PostgreSQL, Redis, automatic database migrations, the API server, and the Next.js dashboard. All services include health checks and restart policies.

### Kubernetes (Helm)

A Helm chart is provided for k3s/k8s deployments:

```bash
helm upgrade --install openoms deploy/helm/openoms \
  -n openoms \
  --set apiServer.image.tag=latest \
  --set dashboard.image.tag=latest \
  --set migration.image.tag=latest
```

The CI/CD pipeline (`.github/workflows/release.yml`) builds Docker images, scans them with Trivy, and deploys to k3s via Helm on push to `main`. A release fallback workflow also checks merged PRs that touch release-relevant paths and dispatches `release.yml` only if GitHub does not create a release run for the merge commit.

### Infrastructure Requirements

- PostgreSQL 16+
- Redis 7+ (required outside `development` unless `ALLOW_IN_MEMORY_STATE=true` is explicitly set for a single-node self-hosted deployment)
- Reverse proxy (nginx / Caddy / Traefik / ingress-nginx) for TLS termination

The Docker images are stateless and can be deployed behind a load balancer or on single-node setups like k3s.

---

## Roadmap

### Done

- Allegro integration (OAuth2, offers, orders, "Wysylam z Allegro" shipments)
- InPost integration (Paczkomaty, courier, label generation, tracking, webhooks)
- Dashboard with Kanban board, dark mode, and PWA support
- Order, product, and customer management
- Automation rules engine (triggers, conditions, actions, delays)
- Packing station with barcode scanner
- Reports and CSV export
- 2FA/TOTP, RBAC with custom roles
- Carrier rate shopping across all providers
- Product CSV import with preview
- SMS notifications (SMSAPI)
- Marketing automation (Mailchimp)
- Print templates (orders, invoices, shipping labels)
- Onboarding wizard (4-step guided setup for new tenants)
- DHL, DPD, GLS carrier SDK audit and verification
- Erli marketplace SDK rebuild and verification
- Supplier portal (public, token-based)
- Supplier product enrichment and category mapping

### In Progress (code written, needs production testing)

- 7 additional carriers (DHL, DPD, GLS, UPS, Poczta Polska, Orlen Paczka, FedEx)
- 10 marketplaces (Amazon, eBay, WooCommerce, Kaufland, OLX, Empik, Erli, Shoper, PrestaShop, Shopify)
- Repricing engine (4 pricing strategies)
- Multi-warehouse with PZ/WZ/MM documents
- Invoicing (Fakturownia, inFakt, wFirma) + KSeF e-invoicing
- Product variants and bundles
- Multi-currency (NBP exchange rates)

### Planned

- Public SaaS (hosted version)
- Mobile application
- Plugin/extension marketplace
- Customer self-service portal
- BaseLinker data import

---

## Community

- **Discord:** [discord.gg/3Z5hzeH5](https://discord.gg/3Z5hzeH5) — ask questions, report bugs, suggest features
- **Issues:** [GitHub Issues](https://github.com/openoms-org/openoms/issues)
- **Website:** [openoms.org](https://openoms.org)

We're looking for beta testers — Polish e-commerce sellers who want to try OpenOMS on real orders. Join Discord for details.

---

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for full details.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-feature`)
3. Run tests (`task test`) and linting (`task lint`)
4. Open a pull request against `main`

Please open an [issue](https://github.com/openoms-org/openoms/issues) first for large changes or new features.

---

## CI/CD

This project uses a two-repository deployment model:

- **Public repo** (`openoms-org/openoms`): Builds Docker images on every push to `main`, pushes to GHCR, runs Trivy security scans, and has a post-merge fallback that dispatches the release workflow if the normal `push` trigger is missing
- **Enterprise repo** (private): Deploys to production via Helm with environment-specific values overlay

Docker images are public on GHCR:
- `ghcr.io/openoms-org/openoms-api`
- `ghcr.io/openoms-org/openoms-dashboard`
- `ghcr.io/openoms-org/openoms-migrate`

---

## License

- **Core applications** (`apps/`): [Elastic License 2.0](LICENSE)
- **SDK packages** (`packages/`): [MIT License](packages/allegro-go-sdk/LICENSE)

---

Built in Poland. Open by default.
