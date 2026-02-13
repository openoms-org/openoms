# OpenOMS Scaling Roadmap

## Phase 1: Zero-Downtime Deploy (ASAP — before first real client)

- [ ] preStop hook + graceful shutdown (Go SIGTERM handling)
- [ ] PodDisruptionBudgets (minAvailable: 1 for API + dashboard)
- [ ] Pod anti-affinity (spread API/dashboard across nodes)
- [ ] Deploy workflow concurrency lock (prevent parallel Helm upgrades)
- [ ] Backwards-compatible migration strategy (expand/contract)
- [ ] Graceful shutdown in Go server (drain connections on SIGTERM)

## Phase 2: Database Reliability (ASAP)

- [ ] PostgreSQL backup CronJob (pg_dump to S3/MinIO)
- [ ] Point-in-time recovery setup (WAL archiving)

## Phase 3: Database HA (~50 clients)

- [ ] CloudNativePG operator (1 primary + 2 replicas, auto-failover)
- [ ] PgBouncer connection pooling (built into CNPG)
- [ ] Read replicas for dashboard queries and reports
- [ ] Redis Sentinel or KeyDB for Redis HA

## Phase 4: Application Scaling (~200-500 clients)

- [ ] HPA (Horizontal Pod Autoscaler) for API pods
- [ ] Worker scaling per job type (separate workers for email, marketplace sync, etc.)
- [ ] Async report generation (background jobs, download when ready)
- [ ] MinIO cluster mode or migrate to real S3
- [ ] Grafana + Loki monitoring stack

## Phase 5: Architecture (~5000+ clients)

- [ ] Table partitioning by tenant_id (pg_partman)
- [ ] Hybrid tenant isolation (shared for small, dedicated for enterprise)
- [ ] Multi-region deployment
- [ ] Redis → dedicated message broker (NATS/RabbitMQ)
- [ ] Canary deployments (Argo Rollouts)
