# OpenOMS

**Open-source Order Management System for Polish e-commerce.**

OpenOMS is a self-hostable, API-first OMS built for Polish online sellers. Connect your sales channels (Allegro, WooCommerce), manage orders, and automate shipping (InPost, DHL, DPD) from one place — without vendor lock-in.

> **Status: MVP Development** — Building the core system. [Join the waitlist](https://openoms.org) to be the first to test it.

## Why OpenOMS?

Polish e-commerce sellers use closed OMS platforms (Base, Sellasist, Apilo) that control their data, integrations, and pricing. When those platforms raise prices — and they do — sellers have no real alternative.

- **Open source (AGPLv3)** — full code transparency, no vendor lock-in
- **Self-hostable** — run on your own infrastructure for free, or use our managed cloud
- **API-first** — REST API with OpenAPI spec, integrate with anything
- **Built for Poland** — Allegro, InPost, Polish VAT, PLN-first

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.24 |
| Database | PostgreSQL 16 (Row-Level Security) |
| Cache/Queue | Redis 7 + asynq |
| Frontend | React 18 + TypeScript + Tailwind CSS + shadcn/ui |
| Auth | JWT (Ed25519) + bcrypt |
| Deployment | Docker Compose |

## Repository Structure

```
openoms/
├── packages/                    # Standalone libraries (MIT license)
│   ├── allegro-go-sdk/          # Allegro REST API client
│   ├── inpost-go-sdk/           # InPost ShipX API client
│   └── order-engine/            # Order state machine + domain events
├── apps/
│   ├── api-server/              # Main backend (AGPLv3)
│   └── web-dashboard/           # React SPA (AGPLv3)
├── docs/                        # Documentation
├── deploy/                      # Deployment configs
├── docker-compose.dev.yml       # Local development
├── Taskfile.yml                 # Dev commands (task up, task run, etc.)
└── .env.example                 # Environment variables template
```

## Quick Start (Development)

```bash
# Prerequisites: Go 1.24+, Docker, Task (go-task.github.io)

# 1. Clone and setup
git clone https://github.com/openoms-org/openoms.git
cd openoms
cp .env.example .env

# 2. Start infrastructure + seed data
task setup

# 3. Run the API server
task run

# 4. Verify
curl http://localhost:8080/health
```

## Development Commands

```bash
task up          # Start PostgreSQL + Redis
task down        # Stop containers
task migrate     # Run database migrations
task seed        # Load test data (3 tenants, 15 orders)
task run         # Start API server (port 8080)
task test        # Run all tests
task lint        # Run golangci-lint
task clean       # Reset everything (removes volumes)
task setup       # Full setup: up + migrate + seed
```

## Project Status

| Phase | Status |
|-------|--------|
| Phase 0: Research & Validation | Done |
| Phase 1: Monorepo + Database | **In progress** |
| Phase 2: Auth + Users | Planned |
| Phase 3: SDK Packages | Planned |
| Phase 4: API Server | Planned |
| Phase 5: Frontend Dashboard | Planned |
| Phase 6: Integration + Polish | Planned |

## Contributing

We're in early development. You can help by:

- **[Join the waitlist](https://openoms.org)** — be first to test the MVP
- **Star this repo** — helps us gauge interest
- **[Open an issue](https://github.com/openoms-org/openoms/issues)** — share pain points, feature ideas, questions

## License

- **Core (apps/)**: [AGPLv3](LICENSE)
- **SDK Packages (packages/)**: [MIT](packages/allegro-go-sdk/LICENSE)

---

Built in Poland. Open by default.
