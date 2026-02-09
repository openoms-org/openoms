# OpenOMS

**Open-source Order Management System for Polish e-commerce.**

OpenOMS is a self-hostable, API-first OMS built for Polish online sellers. Connect your sales channels (Allegro, BaseLinker, WooCommerce, Shopify), manage orders, inventory, and shipping from one place — without vendor lock-in.

> **Status: Early Stage** — We're validating the idea and building the MVP. The codebase will appear here soon. Want to be notified? [Join the waitlist.](https://openoms.org)

## Why OpenOMS?

Polish e-commerce sellers rely on closed, proprietary platforms that control their data and pricing. When those platforms raise prices or change terms, sellers have no alternative.

OpenOMS is different:

- **Open source (AGPLv3)** — full code transparency, no hidden components
- **Self-hostable** — run it on your own infrastructure, or use our managed cloud
- **API-first** — integrate with anything, automate everything
- **Built for Poland** — Allegro, InPost, Polish VAT, PLN-first

## Planned Features

- Multi-channel order aggregation (Allegro, BaseLinker, WooCommerce, Shopify)
- Real-time inventory sync across all channels
- Shipping automation (InPost, DPD, DHL, Poczta Polska)
- Polish invoicing & VAT compliance
- Webhook-driven architecture
- Docker Compose — one command to deploy

## Tech Stack (Planned)

| Layer | Technology |
|-------|-----------|
| Backend | Python (FastAPI) |
| Database | PostgreSQL |
| Queue | Redis / Celery |
| Frontend | React (dashboard) |
| Deployment | Docker Compose |
| License | AGPLv3 |

## Project Status

See [ROADMAP.md](ROADMAP.md) for the full plan.

| Phase | Status |
|-------|--------|
| Phase 0: Validation | **In progress** |
| Phase 1: MVP | Planned |
| Phase 2: Beta | Planned |
| Phase 3: Public launch | Planned |

## Get Involved

We're not accepting code contributions yet (there's no code to contribute to!), but you can:

- **[Join the waitlist](https://openoms.org)** — be first to test the MVP
- **Star this repo** — helps us gauge interest
- **Open an issue** — share your pain points, feature ideas, or questions

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## Links

- **Website:** [openoms.org](https://openoms.org)
- **License:** [AGPLv3](LICENSE)

---

Built in Poland. Open by default.
