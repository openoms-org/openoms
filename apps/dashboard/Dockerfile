# syntax=docker/dockerfile:1

# ── Dependencies stage ───────────────────────────────────────────
FROM node:22-bookworm-slim AS deps

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci --ignore-scripts

# ── Build stage ──────────────────────────────────────────────────
FROM node:22-bookworm-slim AS builder

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Build-time env vars baked into the Next.js client bundle.
# Do not add NEXT_PUBLIC_OPENOMS_DASHBOARD_SURFACE here. Surface mode is a
# runtime Helm setting (OPENOMS_DASHBOARD_SURFACE) so the public image stays
# client-ready.
ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}

ARG DASHBOARD_API_PROXY_URL
ENV DASHBOARD_API_PROXY_URL=${DASHBOARD_API_PROXY_URL}

ARG NEXT_PUBLIC_INPOST_GEOWIDGET_TOKEN
ENV NEXT_PUBLIC_INPOST_GEOWIDGET_TOKEN=${NEXT_PUBLIC_INPOST_GEOWIDGET_TOKEN}

ARG NEXT_PUBLIC_SENTRY_DSN
ENV NEXT_PUBLIC_SENTRY_DSN=${NEXT_PUBLIC_SENTRY_DSN}

ARG NEXT_PUBLIC_SENTRY_ENVIRONMENT
ENV NEXT_PUBLIC_SENTRY_ENVIRONMENT=${NEXT_PUBLIC_SENTRY_ENVIRONMENT}

ARG SENTRY_RELEASE
ENV SENTRY_RELEASE=${SENTRY_RELEASE}

ARG SENTRY_ORG
ARG SENTRY_PROJECT

# next.config.ts already has output: "standalone"
RUN --mount=type=secret,id=sentry_auth_token \
    export SENTRY_AUTH_TOKEN="$(cat /run/secrets/sentry_auth_token 2>/dev/null || true)" && \
    npx next build

# ── Runtime stage ────────────────────────────────────────────────
# Distroless has no apt. CVE-2026-45447 / DSA-6335-1 needs
# libssl3t64 >= 3.5.6-1~deb13u2. Pin the multi-arch index that ships it;
# Renovate bumps the digest.
FROM gcr.io/distroless/nodejs22-debian13:nonroot@sha256:4e4fb0ce55fd73901600796ef079a9490369d2515d7da31633a91608c82ca13b AS runner

WORKDIR /app

ARG SENTRY_RELEASE
ENV NODE_ENV=production
ENV SENTRY_RELEASE=${SENTRY_RELEASE}

# Copy only the standalone build output. UID/GID 65532 is the distroless nonroot user.
COPY --from=builder --chown=65532:65532 /app/public ./public
COPY --from=builder --chown=65532:65532 /app/.next/standalone ./
COPY --from=builder --chown=65532:65532 /app/.next/static ./.next/static

USER 65532:65532

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["server.js"]
