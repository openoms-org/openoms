# Contributing to OpenOMS

Thanks for your interest in contributing to OpenOMS! This guide will help you get started.

## Prerequisites

- **Go** 1.24+
- **Node.js** 22+
- **Docker** (with Docker Compose)
- **[Task](https://taskfile.dev)** (task runner)

## Development Setup

```bash
git clone https://github.com/openoms-org/openoms.git
cd openoms
cp .env.example .env
task setup   # Start containers, run migrations, seed data
task dev     # Start API + dashboard in parallel
```

The API server runs on `http://localhost:8080` and the dashboard on `http://localhost:3000` by default.

## Project Structure

```
apps/api-server/     Go backend (REST API, integrations, business logic)
apps/dashboard/      Next.js frontend (merchant dashboard)
packages/            Standalone SDKs (e.g. allegro-go-sdk, inpost-go-sdk)
```

- **apps/api-server** -- The core Go service handling orders, inventory, integrations, and webhooks.
- **apps/dashboard** -- A Next.js app providing the merchant-facing UI.
- **packages/** -- Independent Go SDK packages for third-party APIs. Each can be imported separately.

## Code Style

### Go

We use [golangci-lint](https://golangci-lint.run/) with the config in `.golangci.yml`.

```bash
task fmt    # Format Go code
task lint   # Run linter checks
```

### TypeScript

We use ESLint and Prettier in the dashboard app.

```bash
cd apps/dashboard
npm run lint
```

## Testing

```bash
task test              # Run Go tests
cd apps/dashboard && npm run test   # Run Vitest for the dashboard
```

## Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` -- A new feature
- `fix:` -- A bug fix
- `docs:` -- Documentation changes
- `chore:` -- Maintenance tasks (deps, CI, config)
- `refactor:` -- Code restructuring without behavior change
- `test:` -- Adding or updating tests

Examples:

```
feat: add Allegro order sync
fix: prevent duplicate webhook delivery
docs: update self-hosting guide
chore: bump Go to 1.24
```

## Pull Request Process

1. **Fork** the repository.
2. **Create a branch** from `main` (`git checkout -b feat/my-feature`).
3. **Make your changes** and add tests where appropriate.
4. **Run checks** before pushing:
   ```bash
   task fmt
   task lint
   task test
   cd apps/dashboard && npm run lint && npm run build
   ```
5. **Open a PR** against `main` with a clear description of what changed and why.
6. A maintainer will review your PR. Address any feedback and keep the branch up to date.

## Adding a New Integration

OpenOMS uses a factory/registry pattern for marketplace and carrier integrations.

1. Create a new package in `internal/integration/<name>/` with a `provider.go` file.
2. Implement the required provider interface (`MarketplaceProvider` or `CarrierProvider`).
3. In the `init()` function, call `RegisterMarketplaceProvider` or `RegisterCarrierProvider` to register your provider with the factory.
4. If the integration needs an external API client, add a standalone SDK package in `packages/<name>-go-sdk/`.

Look at existing integrations (e.g., `internal/integration/allegro/`) for reference.

## Integration Status

Each integration has one of two statuses:

- **Verified** -- Tested and used in production environments.
- **In Development** -- Implemented but not yet verified in production.

When contributing a new integration, it starts as "In Development" until it has been validated in a real-world setup.

## License

- **apps/** -- Licensed under [AGPLv3](LICENSE)
- **packages/** -- Licensed under [MIT](packages/LICENSE)

If you contribute code, you agree that your contributions will be licensed under the respective license for the directory they belong to.

## Questions?

Open an [issue](https://github.com/openoms-org/openoms/issues) and we will be happy to help.
