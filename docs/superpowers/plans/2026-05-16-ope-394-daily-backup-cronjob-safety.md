# OPE-394 Daily Backup CronJob Safety Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Harden the public Helm daily database backup CronJob so scheduled backups fail closed and upload through the Hetzner-compatible MinIO client path.

**Architecture:** Keep the existing public Helm CronJob shape and `postgres:17-alpine` runtime, but change the shell script to dump to `/tmp/backup.sql`, gzip separately, validate with `gzip -t`, enforce a hard minimum byte size, then upload with Alpine `minio-client`/`mcli`. Retention cleanup should use the same `mcli` alias and must not log credentials.

**Tech Stack:** Helm, Kubernetes CronJob, PostgreSQL `pg_dump`, Alpine `minio-client` (`mcli`), Hetzner Object Storage S3-compatible API.

---

## Scope And Boundaries

- Public repo only for code changes:
  - `deploy/helm/openoms/templates/backup-cronjob.yaml`
  - `deploy/helm/openoms/values.yaml`
  - `docs/system-documentation.md`
  - this plan file
- Enterprise values are read-only validation input. They already set `backup.s3.prefix: "daily/"`, endpoint and retention.
- No live manual Kubernetes backup job in this implementation without a separate explicit operator confirmation.

## Implementation Tasks

### Task 1: Prove Current CronJob Is Unsafe

**Files:**
- Read: `deploy/helm/openoms/templates/backup-cronjob.yaml`

- [x] Render the backup CronJob with `backup.enabled=true` and verify the current script contains `pg_dump ... | gzip` and `aws s3 cp`.
- [x] Save the observed failure condition in the session notes: rendered output lacks `gzip -t`, lacks `mcli cp`, and uploads with Alpine `aws-cli`.

### Task 2: Harden Backup Generation And Upload

**Files:**
- Modify: `deploy/helm/openoms/templates/backup-cronjob.yaml`
- Modify: `deploy/helm/openoms/values.yaml`

- [x] Replace `pg_dump | gzip` with:
  - `pg_dump "${DATABASE_URL}" --no-owner --no-privileges --format=plain --file=/tmp/backup.sql`
  - `gzip -c /tmp/backup.sql > "/tmp/${FILENAME}"`
  - `gzip -t "/tmp/${FILENAME}"`
- [x] Add `MIN_BACKUP_BYTES` env from `backup.minBackupBytes`, default `1000`, and keep the existing hard `exit 1` if the gzip is too small.
- [x] Replace `apk add --no-cache aws-cli` and `aws s3 cp` with:
  - `apk add --no-cache minio-client >/dev/null 2>&1`
  - `export MC_CONFIG_DIR=/tmp/.mc`
  - `mcli alias set openoms-backups "${S3_ENDPOINT}" "${AWS_ACCESS_KEY_ID}" "${AWS_SECRET_ACCESS_KEY}" --api S3v4 --path off`
  - `mcli cp "/tmp/${FILENAME}" "openoms-backups/${S3_BUCKET}/${S3_PREFIX}${FILENAME}"`
- [x] Convert retention cleanup from `aws s3 ls/rm` to `mcli ls/rm`, keeping date-based filename retention and avoiding credential output.

### Task 3: Documentation

**Files:**
- Modify: `docs/system-documentation.md`

- [x] Document that the public daily backup CronJob writes the SQL dump first, validates gzip and minimum size, then uploads using MinIO client to S3-compatible storage.

### Task 4: Verification

**Commands:**
- `helm template openoms deploy/helm/openoms --set backup.enabled=true --set backup.s3.endpoint=https://fsn1.your-objectstorage.com --show-only templates/backup-cronjob.yaml`
- `helm lint deploy/helm/openoms`
- `helm template openoms deploy/helm/openoms > /tmp/openoms-ope-394-rendered.yaml`
- `helm template openoms public/deploy/helm/openoms -f enterprise/deploy/helm/values-production.yaml --show-only templates/backup-cronjob.yaml`
- `git diff --check`
- `./scripts/local-ci.sh`

**Expected evidence:**
- Rendered CronJob contains `pg_dump ... --file=/tmp/backup.sql`, `gzip -t`, `MIN_BACKUP_BYTES`, `mcli alias set`, and `mcli cp`.
- Rendered CronJob does not contain `pg_dump ... | gzip`, `apk add --no-cache aws-cli`, or `aws s3 cp`.
- Helm lint/template and full local CI pass before push.

## Risk And Rollback

- Risk: typo in shell script can break scheduled backups. Mitigation: static render checks, Helm lint/template with production values, and optional post-merge manual CronJob trigger only with operator confirmation.
- Risk: retention cleanup parsing can delete wrong objects. Mitigation: keep deletion restricted to filenames matching `openoms-YYYYMMDD-*.sql.gz` under the configured prefix.
- Rollback: revert the public Helm chart commit or temporarily disable `backup.enabled` in values while retaining pre-deploy backup protection in enterprise deploy.

## Self-Review

- Spec coverage: removes `pg_dump | gzip`, adds gzip validation, keeps hard size failure, replaces Alpine `aws-cli`, avoids secret logging, and updates docs.
- Placeholder scan: no TBD/TODO placeholders.
- Command consistency: `openoms-backups` is consistently the `mcli` alias; object paths use `<alias>/<bucket>/<prefix><file>`.
