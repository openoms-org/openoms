# OpenOMS

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.24-00ADD8.svg)](https://go.dev/)
[![Build](https://img.shields.io/github/actions/workflow/status/openoms-org/openoms/ci.yml?branch=main&label=CI)](https://github.com/openoms-org/openoms/actions)

**Open-source Order Management System for e-commerce.**

OpenOMS is a self-hostable, multi-tenant OMS with 296 API endpoints, 81 dashboard pages, and integrations with 8 marketplaces and 8 carriers. Built with Go and Next.js, designed for teams that need full control over their order operations.

> **Status: Active Development**

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

### Products and Inventory
- Product variants and bundles
- B2B pricing tiers
- Multi-warehouse stock management
- Warehouse documents (PZ/WZ/MM)
- Stocktaking (inventory counts)

### Marketplace Integrations (8)
- **Allegro** -- OAuth2, full offer management, listing creation
- **Amazon SP-API** -- orders and catalog sync
- **WooCommerce** -- bidirectional order and product sync
- **eBay** -- order import and listing management
- **Kaufland** -- marketplace integration
- **OLX** -- listing and order management
- **Mirakl / Empik** -- marketplace connector
- **Erli** -- Polish marketplace integration

### Carrier Integrations (8)
- **InPost** -- Paczkomaty (parcel lockers) + courier
- **DHL** -- domestic and international shipping
- **DPD** -- parcel shipping
- **GLS** -- parcel shipping
- **UPS** -- domestic and international shipping
- **Poczta Polska** -- national postal service
- **Orlen Paczka** -- parcel lockers
- **FedEx** -- international shipping
- Carrier rate shopping across all providers

### Other Integrations
- **Fakturownia** -- invoice generation
- **KSeF** -- Polish national e-invoicing system
- **Mailchimp** -- marketing automation
- **Freshdesk** -- helpdesk tickets
- **SMSAPI / Twilio** -- SMS notifications
- **OpenAI** -- AI product categorization and descriptions
- **NBP** -- exchange rates (multi-currency support)

### Platform
- Multi-tenant SaaS with PostgreSQL Row-Level Security
- 296 REST API endpoints with OpenAPI 3.1 spec (Swagger UI)
- 81 dashboard pages with dark mode, PWA support, keyboard shortcuts
- RBAC with custom roles
- 2FA / TOTP authentication
- WebSocket real-time updates
- Outgoing webhooks (HMAC-SHA256 signed)
- Audit log
- Self-service returns portal
- Prometheus metrics (Bearer token auth)
- Security headers (CSP, X-Frame-Options, HSTS, Referrer-Policy)
- Kubernetes secrets encryption at rest, audit logging

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.24, chi/v5 router, pgx/v5 |
| Frontend | Next.js 16, React 19, TypeScript |
| Styling | Tailwind CSS v4, shadcn/ui |
| State / Data | Zustand, React Query, Zod v4 |
| Charts | Recharts |
| Database | PostgreSQL 16 (Row-Level Security) |
| Cache / Queue | Redis 7, asynq |
| Auth | Ed25519 JWT, bcrypt, TOTP |
| API Spec | OpenAPI 3.1, Swagger UI |
| E2E Tests | Playwright (12 specs) |
| CI/CD | GitHub Actions (lint, test, security scan, auto-format, Trivy) |
| Deployment | Docker Compose (dev + prod), Helm chart (k3s/k8s) |
| Monitoring | Prometheus metrics (token-protected) |

### Codebase at a Glance

| Metric | Count |
|---|---|
| Go source files | 386 (71 test files) |
| TypeScript / TSX files | 234 |
| SQL migrations | 46 |
| API endpoints | 296 |
| Dashboard pages | 81 |
| React components | 81 |
| Custom hooks | 45 |
| Handlers / Services / Repos | 57 / 38 / 28 |
| Background workers | 14 |
| Middleware | 12 |
| SDK packages | 21 |

---

## Quick Start (Development)

**Prerequisites:** Go 1.24+, Node.js 22+, Docker, [Task](https://taskfile.dev)

```bash
# 1. Clone
git clone https://github.com/openoms-org/openoms.git
cd openoms

# 2. Configure environment
cp .env.example .env

# 3. Start infrastructure, run migrations, seed data
task setup

# 4. Start the API server (port 8080)
task run

# 5. In a second terminal -- start the dashboard (port 3000)
task dev
```

Verify the API is running:

```bash
curl http://localhost:8080/health
```

### Development Commands

```bash
task setup       # Full setup: containers + migrations + seed
task up          # Start PostgreSQL + Redis containers
task down        # Stop containers
task run         # Start API server (port 8080)
task dashboard   # Start dashboard dev server (port 3000)
task dev         # Start API server + dashboard in parallel
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
│   ├── api-server/              # Go backend (AGPLv3)
│   │   ├── cmd/server/          # Entrypoint
│   │   ├── internal/            # Handlers, services, repositories, workers
│   │   └── migrations/          # 46 SQL migrations
│   └── dashboard/               # Next.js frontend (AGPLv3)
│       └── src/
├── packages/                    # 21 standalone SDK libraries (MIT)
├── deploy/
│   └── helm/openoms/          # Helm chart for k3s/k8s
├── docs/
│   └── system-documentation.md
├── docker-compose.dev.yml
├── docker-compose.prod.yml
├── Taskfile.yml
├── .github/workflows/
│   ├── ci.yml                 # Lint, test, security scan, auto-format
│   └── deploy.yml             # Build images, Trivy scan, Helm deploy
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

The CI/CD pipeline (`.github/workflows/deploy.yml`) builds Docker images, scans them with Trivy, and deploys to k3s via Helm on push to `main`.

### Infrastructure Requirements

- PostgreSQL 16+
- Redis 7+
- Reverse proxy (nginx / Caddy / Traefik / ingress-nginx) for TLS termination

The Docker images are stateless and can be deployed behind a load balancer or on single-node setups like k3s.

---

## Contributing

Contributions are welcome. To get started:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-feature`)
3. Run tests (`task test`) and linting (`task lint`)
4. Open a pull request against `main`

Please open an [issue](https://github.com/openoms-org/openoms/issues) first for large changes or new features.

---

## License

- **Core applications** (`apps/`): [GNU Affero General Public License v3.0](LICENSE)
- **SDK packages** (`packages/`): [MIT License](packages/allegro-go-sdk/LICENSE)

---

Built in Poland. Open by default.
